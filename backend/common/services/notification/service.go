// Package notification is the notifications collection access.
package notification

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
	// ErrNotFound is returned when the notification does not exist for the user.
	ErrNotFound = errors.New("notification not found")
	// ErrDuplicate is returned when a notification with the same dedupe key exists.
	ErrDuplicate = errors.New("notification already sent")
)

func col() *mongo.Collection {
	return database.GetDBConnection().Collection(models.NotificationsCollection)
}

// Indexes declares the notifications indexes. The dedupe index is unique
// and partial so notifications without a key never collide.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.NotificationsCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "readAt", Value: 1}}},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "dedupeKey", Value: 1}},
			Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"dedupeKey": bson.M{"$exists": true}})},
	}}
}

// EnsureIndexes builds the notifications indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Service reads and writes notifications.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// Insert stores a notification. A duplicate dedupe key returns ErrDuplicate.
func (s *Service) Insert(ctx context.Context, n *models.Notification) error {
	n.CreatedAt = time.Now().UTC()
	res, err := col().InsertOne(ctx, n)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return ErrDuplicate
		}
		return err
	}
	n.ID = res.InsertedID.(bson.ObjectID)
	return nil
}

// List returns a page of notifications, newest first, plus the total.
func (s *Service) List(ctx context.Context, uid bson.ObjectID, page, pageSize int) ([]models.Notification, int64, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	filter := bson.M{"userId": uid}
	total, err := col().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := col().Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((page-1)*pageSize)).SetLimit(int64(pageSize)))
	if err != nil {
		return nil, 0, err
	}
	var out []models.Notification
	if err := cur.All(ctx, &out); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []models.Notification{}
	}
	return out, total, nil
}

// UnreadCount returns how many notifications are unread.
func (s *Service) UnreadCount(ctx context.Context, uid bson.ObjectID) (int64, error) {
	return col().CountDocuments(ctx, bson.M{"userId": uid, "readAt": nil})
}

// MarkRead marks one notification read and returns it.
func (s *Service) MarkRead(ctx context.Context, uid, id bson.ObjectID) (*models.Notification, error) {
	var n models.Notification
	err := col().FindOneAndUpdate(ctx,
		bson.M{"_id": id, "userId": uid},
		bson.A{bson.M{"$set": bson.M{"readAt": bson.M{"$ifNull": bson.A{"$readAt", time.Now().UTC()}}}}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&n)
	if err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &n, nil
}

// MarkAllRead marks every unread notification read and returns the count.
func (s *Service) MarkAllRead(ctx context.Context, uid bson.ObjectID) (int64, error) {
	res, err := col().UpdateMany(ctx, bson.M{"userId": uid, "readAt": nil}, bson.M{"$set": bson.M{"readAt": time.Now().UTC()}})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// DeleteAllForUser removes every notification of a user (account deletion).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := col().DeleteMany(ctx, bson.M{"userId": uid})
	return err
}
