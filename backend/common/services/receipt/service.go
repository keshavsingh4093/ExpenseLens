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

// ---- Admin portal helpers ----

// FindByID loads a receipt regardless of owner (admin/ops).
func (s *Service) FindByID(ctx context.Context, id bson.ObjectID) (*models.Receipt, error) {
	var rc models.Receipt
	if err := col().FindOne(ctx, bson.M{"_id": id}).Decode(&rc); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rc, nil
}

// ReassignCategoryAll moves every user's receipts from one category to
// another (system category deletion).
func (s *Service) ReassignCategoryAll(ctx context.Context, from, to bson.ObjectID) (int64, error) {
	res, err := col().UpdateMany(ctx, bson.M{"categoryId": from},
		bson.M{"$set": bson.M{"categoryId": to, "updatedAt": time.Now().UTC()}})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// CountByCategory returns how many receipts (all users) use the category.
func (s *Service) CountByCategory(ctx context.Context, id bson.ObjectID) (int64, error) {
	return col().CountDocuments(ctx, bson.M{"categoryId": id})
}

// ExtractionDay is one day of extraction volume.
type ExtractionDay struct {
	Date       string `bson:"_id" json:"date"`
	Total      int64  `bson:"total" json:"total"`
	Failed     int64  `bson:"failed" json:"failed"`
	Processing int64  `bson:"processing" json:"processing"`
}

// ExtractionModel is the per-model breakdown.
type ExtractionModel struct {
	Model         string  `bson:"_id" json:"model"`
	Total         int64   `bson:"total" json:"total"`
	Failed        int64   `bson:"failed" json:"failed"`
	AvgConfidence float64 `bson:"avgConfidence" json:"avgConfidence"`
}

// ExtractionTotals summarises a window.
type ExtractionTotals struct {
	Total         int64   `bson:"total" json:"total"`
	Failed        int64   `bson:"failed" json:"failed"`
	Processing    int64   `bson:"processing" json:"processing"`
	Succeeded     int64   `bson:"succeeded" json:"succeeded"`
	AvgConfidence float64 `bson:"avgConfidence" json:"avgConfidence"`
	AvgAttempts   float64 `bson:"avgAttempts" json:"avgAttempts"`
}

var (
	isFailed     = bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "failed"}}, 1, 0}}
	isProcessing = bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", "processing"}}, 1, 0}}
	// confidence only counts for successful extractions
	okConfidence = bson.M{"$cond": bson.A{
		bson.M{"$and": bson.A{bson.M{"$ne": bson.A{"$status", "failed"}}, bson.M{"$gt": bson.A{"$extracted.confidence", 0}}}},
		"$extracted.confidence", nil,
	}}
)

// ExtractionByDay groups receipts created since the given time by UTC day.
func (s *Service) ExtractionByDay(ctx context.Context, since time.Time) ([]ExtractionDay, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{
			"_id":        bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
			"total":      bson.M{"$sum": 1},
			"failed":     bson.M{"$sum": isFailed},
			"processing": bson.M{"$sum": isProcessing},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var out []ExtractionDay
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ExtractionDay{}
	}
	return out, nil
}

// ExtractionByModel groups receipts created since the given time by the
// model that produced (or failed) the extraction.
func (s *Service) ExtractionByModel(ctx context.Context, since time.Time) ([]ExtractionModel, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": since}, "extracted.model": bson.M{"$exists": true}}}},
		{{Key: "$group", Value: bson.M{
			"_id":           "$extracted.model",
			"total":         bson.M{"$sum": 1},
			"failed":        bson.M{"$sum": isFailed},
			"avgConfidence": bson.M{"$avg": okConfidence},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}}}},
	})
	if err != nil {
		return nil, err
	}
	var out []ExtractionModel
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ExtractionModel{}
	}
	return out, nil
}

// ExtractionTotalsSince summarises receipts created since the given time.
func (s *Service) ExtractionTotalsSince(ctx context.Context, since time.Time) (*ExtractionTotals, error) {
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"createdAt": bson.M{"$gte": since}}}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"total":         bson.M{"$sum": 1},
			"failed":        bson.M{"$sum": isFailed},
			"processing":    bson.M{"$sum": isProcessing},
			"avgConfidence": bson.M{"$avg": okConfidence},
			"avgAttempts":   bson.M{"$avg": "$extracted.attempts"},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []ExtractionTotals
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	t := ExtractionTotals{}
	if len(rows) > 0 {
		t = rows[0]
	}
	t.Succeeded = t.Total - t.Failed - t.Processing
	return &t, nil
}

// FailedRow is one failed extraction for the admin list.
type FailedRow struct {
	ID         bson.ObjectID `bson:"_id" json:"id"`
	UserID     bson.ObjectID `bson:"userId" json:"userId"`
	UserEmail  string        `bson:"-" json:"userEmail"`
	Merchant   string        `bson:"merchant" json:"merchant"`
	ImageCount int           `bson:"imageCount" json:"imageCount"`
	Error      string        `bson:"error" json:"error"`
	Model      string        `bson:"model" json:"model"`
	Attempts   int           `bson:"attempts" json:"attempts"`
	FailedAt   time.Time     `bson:"failedAt" json:"failedAt"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt"`
}

// ListFailed pages receipts in the failed state, newest failure first.
func (s *Service) ListFailed(ctx context.Context, page, pageSize int) ([]FailedRow, int64, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	filter := bson.M{"status": "failed"}
	total, err := col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := col().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$sort", Value: bson.D{{Key: "updatedAt", Value: -1}}}},
		{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
		{{Key: "$limit", Value: int64(pageSize)}},
		{{Key: "$project", Value: bson.M{
			"userId": 1, "merchant": 1, "createdAt": 1,
			"imageCount": bson.M{"$size": "$images"},
			"error":      "$extracted.error", "model": "$extracted.model", "attempts": "$extracted.attempts",
			"failedAt": bson.M{"$ifNull": bson.A{"$extracted.extractedAt", "$updatedAt"}},
		}}},
	})
	if err != nil {
		return nil, 0, err
	}
	var rows []FailedRow
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []FailedRow{}
	}
	return rows, total, nil
}
