// Package adminaudit is the admin_audit_log collection access. The admin
// backend's services/audit helper decides what to write; this package only
// persists and lists entries.
package adminaudit

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

func col() *mongo.Collection {
	return database.GetDBConnection().Collection(models.AdminAuditLogCollection)
}

// Indexes declares the audit log indexes: newest first, and per target.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.AdminAuditLogCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "at", Value: -1}}},
		{Keys: bson.D{{Key: "targetId", Value: 1}, {Key: "at", Value: -1}}},
		{Keys: bson.D{{Key: "adminId", Value: 1}, {Key: "at", Value: -1}}},
		{Keys: bson.D{{Key: "action", Value: 1}, {Key: "at", Value: -1}}},
	}}
}

// EnsureIndexes builds the indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Service reads and writes audit entries.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// Insert stores one entry, stamping At when unset.
func (s *Service) Insert(ctx context.Context, e *models.AdminAuditEntry) error {
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	res, err := col().InsertOne(ctx, e)
	if err != nil {
		return err
	}
	e.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// ListFilter drives List.
type ListFilter struct {
	Action   string
	AdminID  *bson.ObjectID
	TargetID string
	Page     int
	PageSize int
}

// Normalize clamps paging.
func (f *ListFilter) Normalize() {
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 25
	}
	if f.Page < 1 {
		f.Page = 1
	}
}

// List returns a page of entries, newest first, plus the total count.
func (s *Service) List(ctx context.Context, f ListFilter) ([]models.AdminAuditEntry, int64, error) {
	f.Normalize()
	filter := bson.M{}
	if f.Action != "" {
		filter["action"] = f.Action
	}
	if f.AdminID != nil {
		filter["adminId"] = *f.AdminID
	}
	if f.TargetID != "" {
		filter["targetId"] = f.TargetID
	}
	total, err := col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := col().Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "at", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64((f.Page-1)*f.PageSize)).SetLimit(int64(f.PageSize)))
	if err != nil {
		return nil, 0, err
	}
	var out []models.AdminAuditEntry
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []models.AdminAuditEntry{}
	}
	return out, total, nil
}

// Actions returns the distinct action names present (filter dropdown).
func (s *Service) Actions(ctx context.Context) ([]string, error) {
	raw, err := col().Distinct(ctx, "action", bson.M{}).Raw()
	if err != nil {
		return nil, err
	}
	vals, err := raw.Values()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.StringValueOK(); ok {
			out = append(out, s)
		}
	}
	return out, nil
}
