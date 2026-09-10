package receipts

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"expenselens/backend/internal/db"
)

// ErrNotFound is returned when the receipt does not exist for the user.
var ErrNotFound = errors.New("receipt not found")

// Repo is the receipts collection.
type Repo struct{ col *mongo.Collection }

// NewRepo binds to "receipts".
func NewRepo(d *db.DB) *Repo { return &Repo{col: d.Collection("receipts")} }

// Indexes declares receipt indexes.
func Indexes() db.IndexSet {
	return db.IndexSet{Collection: "receipts", Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "categoryId", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "fileHash", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "merchantNormalized", Value: 1}}},
	}}
}

// NormalizeMerchant lowercases and collapses whitespace for grouping/search.
func NormalizeMerchant(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

func (r *Repo) Insert(ctx context.Context, rc *Receipt) error {
	now := time.Now().UTC()
	rc.CreatedAt, rc.UpdatedAt = now, now
	rc.MerchantNormalized = NormalizeMerchant(rc.Merchant)
	if rc.Items == nil {
		rc.Items = []Item{}
	}
	if rc.Tags == nil {
		rc.Tags = []string{}
	}
	if rc.Images == nil {
		rc.Images = []Image{}
	}
	res, err := r.col.InsertOne(ctx, rc)
	if err != nil {
		return err
	}
	rc.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

func (r *Repo) Get(ctx context.Context, uid, id bson.ObjectID) (*Receipt, error) {
	var rc Receipt
	if err := r.col.FindOne(ctx, bson.M{"_id": id, "userId": uid}).Decode(&rc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rc, nil
}

// Update applies $set (and optional $unset) for the user's receipt.
func (r *Repo) Update(ctx context.Context, uid, id bson.ObjectID, set bson.M, unset ...string) error {
	set["updatedAt"] = time.Now().UTC()
	if m, ok := set["merchant"].(string); ok {
		set["merchantNormalized"] = NormalizeMerchant(m)
	}
	upd := bson.M{"$set": set}
	if len(unset) > 0 {
		u := bson.M{}
		for _, k := range unset {
			u[k] = ""
		}
		upd["$unset"] = u
	}
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": id, "userId": uid}, upd)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "userId": uid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByHash finds a receipt with identical uploaded bytes.
func (r *Repo) FindByHash(ctx context.Context, uid bson.ObjectID, hash string) (*Receipt, error) {
	var rc Receipt
	err := r.col.FindOne(ctx, bson.M{"userId": uid, "fileHash": hash}).Decode(&rc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rc, nil
}

// ListFilter drives List.
type ListFilter struct {
	Query      string
	CategoryID *bson.ObjectID
	Status     string
	From, To   string // YYYY-MM-DD inclusive
	Sort       string // date_desc (default), date_asc, amount_desc, amount_asc, created_desc
	Page       int
	PageSize   int
}

// List returns a page of compact rows plus the total count.
func (r *Repo) List(ctx context.Context, uid bson.ObjectID, f ListFilter) ([]ListItem, int64, error) {
	filter := bson.M{"userId": uid}
	if f.CategoryID != nil {
		filter["categoryId"] = *f.CategoryID
	}
	if f.Status != "" {
		filter["status"] = f.Status
	}
	if f.From != "" || f.To != "" {
		d := bson.M{}
		if f.From != "" {
			d["$gte"] = f.From
		}
		if f.To != "" {
			d["$lte"] = f.To
		}
		filter["date"] = d
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		re := bson.Regex{Pattern: regexp.QuoteMeta(q), Options: "i"}
		filter["$or"] = bson.A{
			bson.M{"merchant": re}, bson.M{"items.name": re}, bson.M{"notes": re}, bson.M{"tags": re},
		}
	}
	sort := bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}
	switch f.Sort {
	case "date_asc":
		sort = bson.D{{Key: "date", Value: 1}, {Key: "createdAt", Value: 1}}
	case "amount_desc":
		sort = bson.D{{Key: "total", Value: -1}}
	case "amount_asc":
		sort = bson.D{{Key: "total", Value: 1}}
	case "created_desc":
		sort = bson.D{{Key: "createdAt", Value: -1}}
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := r.col.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$sort", Value: sort}},
		{{Key: "$skip", Value: int64((f.Page - 1) * f.PageSize)}},
		{{Key: "$limit", Value: int64(f.PageSize)}},
		{{Key: "$project", Value: bson.M{
			"status": 1, "merchant": 1, "date": 1, "categoryId": 1, "currency": 1, "total": 1, "createdAt": 1,
			"itemCount": bson.M{"$size": "$items"},
			"firstKey":  bson.M{"$arrayElemAt": bson.A{"$images.key", 0}},
		}}},
	})
	if err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ListItem `bson:",inline"`
		FirstKey string `bson:"firstKey"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, err
	}
	out := make([]ListItem, 0, len(rows))
	for _, row := range rows {
		li := row.ListItem
		li.firstKey = row.FirstKey
		out = append(out, li)
	}
	return out, total, nil
}

// Recent returns the newest uploads regardless of status.
func (r *Repo) Recent(ctx context.Context, uid bson.ObjectID, n int) ([]Receipt, error) {
	cur, err := r.col.Find(ctx, bson.M{"userId": uid},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(n)))
	if err != nil {
		return nil, err
	}
	var out []Receipt
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Receipt{}
	}
	return out, nil
}

// ReassignCategory moves all receipts from one category to another.
func (r *Repo) ReassignCategory(ctx context.Context, uid, from, to bson.ObjectID) (int64, error) {
	res, err := r.col.UpdateMany(ctx, bson.M{"userId": uid, "categoryId": from},
		bson.M{"$set": bson.M{"categoryId": to, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// AllImageKeys returns every storage key for the user (account deletion).
func (r *Repo) AllImageKeys(ctx context.Context, uid bson.ObjectID) ([]string, error) {
	cur, err := r.col.Find(ctx, bson.M{"userId": uid}, options.Find().SetProjection(bson.M{"images.key": 1}))
	if err != nil {
		return nil, err
	}
	var docs []struct {
		Images []struct {
			Key string `bson:"key"`
		} `bson:"images"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	var keys []string
	for _, d := range docs {
		for _, im := range d.Images {
			keys = append(keys, im.Key)
		}
	}
	return keys, nil
}

// DeleteAllForUser removes all receipts for uid.
func (r *Repo) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"userId": uid})
	return err
}
