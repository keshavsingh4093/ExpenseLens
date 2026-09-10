// Package database owns the MongoDB client. Configure once, Connect at boot,
// then services reach the database through GetDBConnection.
package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoConfig configures the connection.
type MongoConfig struct {
	URI         string
	Database    string
	MaxPoolSize uint64
}

var (
	cfg    = MongoConfig{URI: "mongodb://localhost:27017", Database: "expenselens", MaxPoolSize: 50}
	client *mongo.Client
	db     *mongo.Database
)

// ConfigureMongo sets connection parameters. Call before Connect.
func ConfigureMongo(c MongoConfig) {
	if c.URI != "" {
		cfg.URI = c.URI
	}
	if c.Database != "" {
		cfg.Database = c.Database
	}
	if c.MaxPoolSize > 0 {
		cfg.MaxPoolSize = c.MaxPoolSize
	}
}

// Connect opens the client and pings the server.
func Connect(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	c, err := mongo.Connect(options.Client().ApplyURI(cfg.URI).SetMaxPoolSize(cfg.MaxPoolSize).SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	if err := c.Ping(ctx, nil); err != nil {
		_ = c.Disconnect(context.Background())
		return fmt.Errorf("mongo ping: %w", err)
	}
	client = c
	db = c.Database(cfg.Database)
	return nil
}

// Disconnect closes the client.
func Disconnect(ctx context.Context) error {
	if client == nil {
		return nil
	}
	err := client.Disconnect(ctx)
	client, db = nil, nil
	return err
}

// GetDBConnection returns the application database. Panics if Connect has
// not run: every caller is a service that only exists after boot.
func GetDBConnection() *mongo.Database {
	if db == nil {
		panic("database: Connect has not been called")
	}
	return db
}

// GetClient returns the underlying client (sessions/transactions).
func GetClient() *mongo.Client { return client }

// IsDuplicateKey reports whether err is a unique-index violation.
func IsDuplicateKey(err error) bool { return mongo.IsDuplicateKeyError(err) }

// IsNoDocuments reports whether err is mongo.ErrNoDocuments.
func IsNoDocuments(err error) bool { return errors.Is(err, mongo.ErrNoDocuments) }

// IndexSet describes the indexes for one collection.
type IndexSet struct {
	Collection string
	Models     []mongo.IndexModel
}

// EnsureIndexes creates every index idempotently. It must only be called
// from tooling (cmd/scripts/ensure-indexes), never on the server boot path.
func EnsureIndexes(ctx context.Context, sets ...IndexSet) error {
	d := GetDBConnection()
	for _, s := range sets {
		if len(s.Models) == 0 {
			continue
		}
		if _, err := d.Collection(s.Collection).Indexes().CreateMany(ctx, s.Models); err != nil {
			return fmt.Errorf("indexes for %s: %w", s.Collection, err)
		}
	}
	return nil
}

func keySignature(keys any) string {
	raw, err := bson.Marshal(keys)
	if err != nil {
		return ""
	}
	var d bson.D
	if err := bson.Unmarshal(raw, &d); err != nil {
		return ""
	}
	parts := make([]string, 0, len(d))
	for _, e := range d {
		parts = append(parts, fmt.Sprintf("%s:%v", e.Key, e.Value))
	}
	return strings.Join(parts, ",")
}

// MissingIndexes reports declared indexes that do not exist yet, as
// "collection{key:1,...}" strings. Used at boot to warn without building.
func MissingIndexes(ctx context.Context, sets ...IndexSet) ([]string, error) {
	d := GetDBConnection()
	var missing []string
	for _, s := range sets {
		if len(s.Models) == 0 {
			continue
		}
		cur, err := d.Collection(s.Collection).Indexes().List(ctx)
		if err != nil {
			return nil, err
		}
		var existing []bson.M
		if err := cur.All(ctx, &existing); err != nil {
			return nil, err
		}
		have := map[string]bool{}
		for _, ix := range existing {
			have[keySignature(ix["key"])] = true
		}
		for _, m := range s.Models {
			if sig := keySignature(m.Keys); !have[sig] {
				missing = append(missing, s.Collection+"{"+sig+"}")
			}
		}
	}
	return missing, nil
}
