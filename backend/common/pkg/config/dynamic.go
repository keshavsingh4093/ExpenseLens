package config

import (
	"context"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

// SystemConfigCollection is the Mongo collection holding dynamic keys.
const SystemConfigCollection = models.SystemConfigCollection

type dynamicConfig struct {
	mu     sync.RWMutex
	values map[string]string
}

var dynamic *dynamicConfig

// InitDynamicConfig loads every system_config document into memory. Call
// after database.Connect. Failure leaves the cache empty (env/defaults win).
func InitDynamicConfig(ctx context.Context) error {
	dynamic = &dynamicConfig{values: map[string]string{}}
	return ReloadDynamicConfig(ctx)
}

// GetDynamic returns a cached dynamic value, or fallback when missing/blank.
func GetDynamic(key, fallback string) string {
	if dynamic == nil {
		return fallback
	}
	dynamic.mu.RLock()
	defer dynamic.mu.RUnlock()
	if v, ok := dynamic.values[key]; ok && v != "" {
		return v
	}
	return fallback
}

// ReloadDynamicConfig re-reads the collection into the cache.
func ReloadDynamicConfig(ctx context.Context) error {
	if dynamic == nil {
		dynamic = &dynamicConfig{values: map[string]string{}}
	}
	cur, err := database.GetDBConnection().Collection(SystemConfigCollection).Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	var docs []models.SystemConfig
	if err := cur.All(ctx, &docs); err != nil {
		return err
	}
	next := make(map[string]string, len(docs))
	for _, d := range docs {
		next[d.Key] = d.Value
	}
	dynamic.mu.Lock()
	dynamic.values = next
	dynamic.mu.Unlock()
	return nil
}

// SystemConfigIndexes declares the unique key index (built by ensure-indexes).
func SystemConfigIndexes() database.IndexSet {
	return database.IndexSet{Collection: SystemConfigCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)},
	}}
}
