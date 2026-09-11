// Package warranty is the warranties collection access.
package warranty

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

// ErrNotFound is returned when the warranty does not exist for the user.
var ErrNotFound = errors.New("warranty not found")

func col() *mongo.Collection {
	return database.GetDBConnection().Collection(models.WarrantiesCollection)
}

// Indexes declares the warranties indexes.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.WarrantiesCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "expiryDate", Value: 1}}},
		{Keys: bson.D{{Key: "expiryDate", Value: 1}}},
	}}
}

// EnsureIndexes builds the warranties indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Service reads and writes warranties.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// Insert stores a warranty.
func (s *Service) Insert(ctx context.Context, w *models.Warranty) error {
	now := time.Now().UTC()
	w.CreatedAt, w.UpdatedAt = now, now
	res, err := col().InsertOne(ctx, w)
	if err != nil {
		return err
	}
	w.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// List returns the user's warranties, soonest expiry first.
func (s *Service) List(ctx context.Context, uid bson.ObjectID) ([]models.Warranty, error) {
	cur, err := col().Find(ctx, bson.M{"userId": uid}, options.Find().SetSort(bson.D{{Key: "expiryDate", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []models.Warranty
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.Warranty{}
	}
	return out, nil
}

// Get loads one warranty owned by uid.
func (s *Service) Get(ctx context.Context, uid, id bson.ObjectID) (*models.Warranty, error) {
	var w models.Warranty
	if err := col().FindOne(ctx, bson.M{"_id": id, "userId": uid}).Decode(&w); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

// Update applies a $set patch (and optional $unset) for the user's warranty.
func (s *Service) Update(ctx context.Context, uid, id bson.ObjectID, set bson.M, unset ...string) error {
	set["updatedAt"] = time.Now().UTC()
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

// Delete removes one warranty owned by uid.
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

// ExpiringBetween returns warranties (all users) expiring in [from, to].
func (s *Service) ExpiringBetween(ctx context.Context, from, to string) ([]models.Warranty, error) {
	cur, err := col().Find(ctx, bson.M{"expiryDate": bson.M{"$gte": from, "$lte": to}})
	if err != nil {
		return nil, err
	}
	var out []models.Warranty
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MarkReminderSent records that the reminder tag ("30"/"7") was sent.
func (s *Service) MarkReminderSent(ctx context.Context, id bson.ObjectID, tag string) error {
	_, err := col().UpdateByID(ctx, id, bson.M{"$addToSet": bson.M{"remindersSent": tag}})
	return err
}

// DeleteAllForUser removes every warranty of a user (account deletion).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := col().DeleteMany(ctx, bson.M{"userId": uid})
	return err
}
