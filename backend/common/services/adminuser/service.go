// Package adminuser is the admin_users collection access.
package adminuser

import (
	"context"
	"errors"
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
		a.Role = "admin"
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
