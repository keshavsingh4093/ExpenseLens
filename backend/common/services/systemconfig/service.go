// Package systemconfig reads and writes the dynamic system_config keys.
package systemconfig

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/config"
	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

// ErrNotFound is returned when the key does not exist.
var ErrNotFound = errors.New("config key not found")

func col() *mongo.Collection {
	return database.GetDBConnection().Collection(models.SystemConfigCollection)
}

// Service manages system_config documents.
type Service struct{}

// New returns the service.
func New() *Service { return &Service{} }

// EnsureIndexes builds the unique key index (tooling only).
func EnsureIndexes(ctx context.Context) error {
	return database.EnsureIndexes(ctx, config.SystemConfigIndexes())
}

// List returns every key, sorted by category then key. Sensitive values are
// masked unless reveal is true.
func (s *Service) List(ctx context.Context, category string, reveal bool) ([]models.SystemConfig, error) {
	filter := bson.M{}
	if category != "" {
		filter["category"] = category
	}
	cur, err := col().Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "category", Value: 1}, {Key: "key", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var out []models.SystemConfig
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []models.SystemConfig{}
	}
	if !reveal {
		for i := range out {
			if out[i].Sensitive && out[i].Value != "" {
				out[i].Value = Mask(out[i].Value)
			}
		}
	}
	return out, nil
}

// Mask hides all but the last four characters of a secret.
func Mask(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return strings.Repeat("*", 8) + v[len(v)-4:]
}

// Get returns one key.
func (s *Service) Get(ctx context.Context, key string) (*models.SystemConfig, error) {
	var c models.SystemConfig
	if err := col().FindOne(ctx, bson.M{"key": key}).Decode(&c); err != nil {
		if database.IsNoDocuments(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// Upsert creates or updates a key and refreshes the in-memory cache.
func (s *Service) Upsert(ctx context.Context, c models.SystemConfig) (*models.SystemConfig, error) {
	c.Key = strings.TrimSpace(c.Key)
	now := time.Now().UTC()
	_, err := col().UpdateOne(ctx, bson.M{"key": c.Key}, bson.M{
		"$set": bson.M{
			"value": c.Value, "description": c.Description, "category": c.Category,
			"sensitive": c.Sensitive, "updatedAt": now,
		},
		"$setOnInsert": bson.M{"createdAt": now},
	}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	_ = config.ReloadDynamicConfig(ctx)
	return s.Get(ctx, c.Key)
}

// Delete removes a key and refreshes the cache.
func (s *Service) Delete(ctx context.Context, key string) error {
	res, err := col().DeleteOne(ctx, bson.M{"key": key})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return config.ReloadDynamicConfig(ctx)
}

// Reload refreshes the in-memory cache from the collection.
func (s *Service) Reload(ctx context.Context) error { return config.ReloadDynamicConfig(ctx) }
