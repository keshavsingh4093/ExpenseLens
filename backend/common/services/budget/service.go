// Package budget is the budgets collection access.
package budget

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

// ErrNotFound is returned when the budget does not exist for the user.
var ErrNotFound = errors.New("budget not found")

func col() *mongo.Collection { return database.GetDBConnection().Collection(models.BudgetsCollection) }

// Indexes declares the budgets indexes.
func Indexes() database.IndexSet {
	return database.IndexSet{Collection: models.BudgetsCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "categoryId", Value: 1}, {Key: "month", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "month", Value: 1}, {Key: "userId", Value: 1}}},
	}}
}

// EnsureIndexes builds the budgets indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, Indexes()) }

// Service reads and writes budgets.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// ListByMonth returns the user's budgets for a month.
func (s *Service) ListByMonth(ctx context.Context, uid bson.ObjectID, month string) ([]models.Budget, error) {
	cur, err := col().Find(ctx, bson.M{"userId": uid, "month": month}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []models.Budget
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.Budget{}
	}
	return out, nil
}

// Get loads one budget owned by uid.
func (s *Service) Get(ctx context.Context, uid, id bson.ObjectID) (*models.Budget, error) {
	var b models.Budget
	if err := col().FindOne(ctx, bson.M{"_id": id, "userId": uid}).Decode(&b); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

// Upsert creates or updates the limit for (user, category, month).
func (s *Service) Upsert(ctx context.Context, uid, categoryID bson.ObjectID, month string, limit int64, currency string) (*models.Budget, error) {
	now := time.Now().UTC()
	var b models.Budget
	err := col().FindOneAndUpdate(ctx,
		bson.M{"userId": uid, "categoryId": categoryID, "month": month},
		bson.M{"$set": bson.M{"limit": limit, "currency": currency, "updatedAt": now}, "$setOnInsert": bson.M{"createdAt": now}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
	).Decode(&b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Delete removes one budget owned by uid.
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

// UsersWithBudgets returns the distinct users that have a budget for month.
func (s *Service) UsersWithBudgets(ctx context.Context, month string) ([]bson.ObjectID, error) {
	raw, err := col().Distinct(ctx, "userId", bson.M{"month": month}).Raw()
	if err != nil {
		return nil, err
	}
	vals, err := raw.Values()
	if err != nil {
		return nil, err
	}
	out := make([]bson.ObjectID, 0, len(vals))
	for _, v := range vals {
		if id, ok := v.ObjectIDOK(); ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// ListAllForUser returns every budget of a user (export).
func (s *Service) ListAllForUser(ctx context.Context, uid bson.ObjectID) ([]models.Budget, error) {
	cur, err := col().Find(ctx, bson.M{"userId": uid}, options.Find().SetSort(bson.D{{Key: "month", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var out []models.Budget
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.Budget{}
	}
	return out, nil
}

// DeleteAllForUser removes every budget of a user (account deletion).
func (s *Service) DeleteAllForUser(ctx context.Context, uid bson.ObjectID) error {
	_, err := col().DeleteMany(ctx, bson.M{"userId": uid})
	return err
}
