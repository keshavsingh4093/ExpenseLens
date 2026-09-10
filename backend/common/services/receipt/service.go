package receipt

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

// ErrNotFound is returned when the receipt does not exist for the user.
var ErrNotFound = errors.New("receipt not found")

// Service is the receipts collection access.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// EnsureIndexes builds the receipt indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Insert stores a new receipt, filling timestamps and empty slices.
func (s *Service) Insert(ctx context.Context, rc *models.Receipt) error {
	now := time.Now().UTC()
	rc.CreatedAt, rc.UpdatedAt = now, now
	rc.MerchantNormalized = models.NormalizeMerchant(rc.Merchant)
	if rc.Items == nil {
		rc.Items = []models.ReceiptItem{}
	}
	if rc.Tags == nil {
		rc.Tags = []string{}
	}
	if rc.Images == nil {
		rc.Images = []models.ReceiptImage{}
	}
	res, err := col().InsertOne(ctx, rc)
	if err != nil {
		return err
	}
	rc.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// Get loads one receipt owned by uid.
func (s *Service) Get(ctx context.Context, uid, id bson.ObjectID) (*models.Receipt, error) {
	var rc models.Receipt
	if err := col().FindOne(ctx, bson.M{"_id": id, "userId": uid}).Decode(&rc); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rc, nil
}

// Update applies $set (and optional $unset) for the user's receipt.
func (s *Service) Update(ctx context.Context, uid, id bson.ObjectID, set bson.M, unset ...string) error {
	set["updatedAt"] = time.Now().UTC()
	if m, ok := set["merchant"].(string); ok {
		set["merchantNormalized"] = models.NormalizeMerchant(m)
	}
	upd := bson.M{"$set": set}
	if len(unset) > 0 {
		u := bson.M{}
		for _, k := range unset {
			u[k] = ""
		}
		upd["$unset"] = u
	}
	res, err := col().UpdateOne(ctx, bson.M{"_id": id, "userId": uid}, upd)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes one receipt owned by uid.
func (s *Service) Delete(ctx context.Context, uid, id bson.ObjectID) error {
	res, err := col().DeleteOne(ctx, bson.M{"_id": id, "userId": uid})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByHash finds a receipt with identical uploaded bytes.
func (s *Service) FindByHash(ctx context.Context, uid bson.ObjectID, hash string) (*models.Receipt, error) {
	var rc models.Receipt
	err := col().FindOne(ctx, bson.M{"userId": uid, "fileHash": hash}).Decode(&rc)
	if err != nil {
		if database.IsNoDocuments(err) {
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

// Normalize clamps paging to sane values.
func (f *ListFilter) Normalize() {
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
}

// List returns a page of compact rows plus the total count.
func (s *Service) List(ctx context.Context, uid bson.ObjectID, f ListFilter) ([]models.ReceiptListItem, int64, error) {
	f.Normalize()
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
	total, err := col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
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
	var rows []models.ReceiptListItem
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []models.ReceiptListItem{}
	}
	return rows, total, nil
}

// Recent returns the newest uploads regardless of status.
func (s *Service) Recent(ctx context.Context, uid bson.ObjectID, n int) ([]models.Receipt, error) {
	cur, err := col().Find(ctx, bson.M{"userId": uid},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(int64(n)))
	if err != nil {
		return nil, err
	}
	var out []models.Receipt
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.Receipt{}
	}
	return out, nil
}

// ReassignCategory moves all receipts from one category to another.
func (s *Service) ReassignCategory(ctx context.Context, uid, from, to bson.ObjectID) (int64, error) {
	res, err := col().UpdateMany(ctx, bson.M{"userId": uid, "categoryId": from},
		bson.M{"$set": bson.M{"categoryId": to, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// AllImageKeys returns every storage key for the user (account deletion).
func (s *Service) AllImageKeys(ctx context.Context, uid bson.ObjectID) ([]string, error) {
	cur, err := col().Find(ctx, bson.M{"userId": uid}, options.Find().SetProjection(bson.M{"images.key": 1}))
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
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := col().DeleteMany(ctx, bson.M{"userId": uid})
	return err
}

// Count returns the number of receipts matching filter (admin stats).
func (s *Service) Count(ctx context.Context, filter bson.M) (int64, error) {
	if filter == nil {
		filter = bson.M{}
	}
	return col().CountDocuments(ctx, filter)
}

// CountByStatus returns receipt counts per status (admin stats).
func (s *Service) CountByStatus(ctx context.Context) (map[string]int64, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.M{"_id": "$status", "n": bson.M{"$sum": 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID string `bson:"_id"`
		N  int64  `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.ID] = r.N
	}
	return out, nil
}

// CountUsersWithReceipts returns how many distinct users own a receipt.
func (s *Service) CountUsersWithReceipts(ctx context.Context) (int64, error) {
	res, err := col().Distinct(ctx, "userId", bson.M{}).Raw()
	if err != nil {
		return 0, err
	}
	vals, err := res.Values()
	if err != nil {
		return 0, err
	}
	return int64(len(vals)), nil
}

// FindAll returns every counted receipt (review/confirmed) for the user in
// the inclusive date window, oldest first, capped at 5000 rows (exports).
func (s *Service) FindAll(ctx context.Context, uid bson.ObjectID, from, to string) ([]models.Receipt, error) {
	filter := bson.M{"userId": uid, "status": bson.M{"$in": bson.A{"review", "confirmed"}}}
	if from != "" || to != "" {
		d := bson.M{}
		if from != "" {
			d["$gte"] = from
		}
		if to != "" {
			d["$lte"] = to
		}
		filter["date"] = d
	}
	cur, err := col().Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "date", Value: 1}, {Key: "createdAt", Value: 1}}).SetLimit(5000))
	if err != nil {
		return nil, err
	}
	var out []models.Receipt
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.Receipt{}
	}
	return out, nil
}

// ListAllForUser returns every receipt of the user regardless of status (exports).
func (s *Service) ListAllForUser(ctx context.Context, uid bson.ObjectID) ([]models.Receipt, error) {
	cur, err := col().Find(ctx, bson.M{"userId": uid}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetLimit(5000))
	if err != nil {
		return nil, err
	}
	var out []models.Receipt
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.Receipt{}
	}
	return out, nil
}
