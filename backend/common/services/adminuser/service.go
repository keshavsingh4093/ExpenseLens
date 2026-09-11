// Package adminuser is the admin_users collection access.
package adminuser

import (
	"context"
	"errors"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

var (
	ErrNotFound   = errors.New("admin not found")
	ErrEmailTaken = errors.New("admin email already exists")
)

func col() *mongo.Collection {
	return database.GetDBConnection().Collection(models.AdminUsersCollection)
}

// Indexes declares the admin_users indexes.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.AdminUsersCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
	}}
}

// EnsureIndexes builds the indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Service reads and writes admin accounts.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// Create inserts an admin. PasswordHash must already be hashed.
func (s *Service) Create(ctx context.Context, a *models.AdminUser) error {
	now := time.Now().UTC()
	a.Email = models.NormalizeEmail(a.Email)
	a.CreatedAt, a.UpdatedAt = now, now
	if a.Role == "" {
		a.Role = models.AdminRoleAdmin
	}
	res, err := col().InsertOne(ctx, a)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return ErrEmailTaken
		}
		return err
	}
	a.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// FindByEmail loads an admin by email.
func (s *Service) FindByEmail(ctx context.Context, email string) (*models.AdminUser, error) {
	return s.findOne(ctx, bson.M{"email": models.NormalizeEmail(email)})
}

// FindByID loads an admin by id.
func (s *Service) FindByID(ctx context.Context, id bson.ObjectID) (*models.AdminUser, error) {
	return s.findOne(ctx, bson.M{"_id": id})
}

func (s *Service) findOne(ctx context.Context, filter bson.M) (*models.AdminUser, error) {
	var a models.AdminUser
	if err := col().FindOne(ctx, filter).Decode(&a); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// Update applies a $set patch.
func (s *Service) Update(ctx context.Context, id bson.ObjectID, set bson.M) error {
	set["updatedAt"] = time.Now().UTC()
	res, err := col().UpdateByID(ctx, id, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// Count returns the number of admin accounts.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return col().CountDocuments(ctx, bson.M{})
}

// CountSuperadmins returns the number of superadmins that are not disabled.
// The admin portal refuses to disable or demote the last one.
func (s *Service) CountSuperadmins(ctx context.Context) (int64, error) {
	return col().CountDocuments(ctx, bson.M{"role": models.AdminRoleSuperadmin, "disabled": bson.M{"$ne": true}})
}

// IsDisabled reports the account flag with a projected lookup (auth guard).
func (s *Service) IsDisabled(ctx context.Context, id bson.ObjectID) (bool, error) {
	var row struct {
		Disabled bool `bson:"disabled"`
	}
	err := col().FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"disabled": 1})).Decode(&row)
	if err != nil {
		if database.IsNoDocuments(err) {
			return false, ErrNotFound
		}
		return false, err
	}
	return row.Disabled, nil
}

// SetDisabled flips the account flag. Callers revoke sessions themselves.
func (s *Service) SetDisabled(ctx context.Context, id bson.ObjectID, disabled bool) error {
	return s.Update(ctx, id, bson.M{"disabled": disabled})
}

// ListFilter drives List (admin portal).
type ListFilter struct {
	Query    string // matches email or name, case-insensitive
	Page     int
	PageSize int
}

// List pages admin accounts, newest first, with the total for the filter.
func (s *Service) List(ctx context.Context, f ListFilter) ([]models.AdminUser, int64, error) {
	filter := bson.M{}
	if q := f.Query; q != "" {
		re := bson.Regex{Pattern: regexp.QuoteMeta(q), Options: "i"}
		filter["$or"] = bson.A{bson.M{"email": re}, bson.M{"name": re}}
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	total, err := col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64((f.Page - 1) * f.PageSize)).
		SetLimit(int64(f.PageSize))
	cur, err := col().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	rows := []models.AdminUser{}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
