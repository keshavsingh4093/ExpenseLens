// Package db owns the MongoDB connection and index bootstrap.
package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// DB wraps the mongo client and the application database handle.
type DB struct {
	Client *mongo.Client
	*mongo.Database
}

// Connect opens a client, pings the server, and returns the database handle.
func Connect(ctx context.Context, uri, name string) (*DB, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	return &DB{Client: client, Database: client.Database(name)}, nil
}

// Close disconnects the client.
func (d *DB) Close(ctx context.Context) error { return d.Client.Disconnect(ctx) }

// IndexSet describes indexes for one collection.
type IndexSet struct {
	Collection string
	Models     []mongo.IndexModel
}

// EnsureIndexes creates all indexes idempotently. Each domain package exposes
// its own IndexSet; main gathers them and calls this at boot.
func (d *DB) EnsureIndexes(ctx context.Context, sets ...IndexSet) error {
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
