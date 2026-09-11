package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/models"
)

// Opaque tokens (refresh, password reset) are random, stored hashed, and
// single-use. TokenStore is shared by the user and admin portals.

// ErrTokenInvalid is returned when a token is unknown, used, or expired.
var ErrTokenInvalid = errors.New("token invalid or expired")

func tokens() *mongo.Collection {
	return database.GetDBConnection().Collection(models.AuthTokensCollection)
}

// TokenIndexes declares token indexes, including a TTL that purges expired rows.
func TokenIndexes() database.IndexSet {
	return database.IndexSet{Collection: models.AuthTokensCollection, Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "hash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "kind", Value: 1}}},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}}
}

// EnsureIndexes builds the token indexes (tooling only).
func EnsureIndexes(ctx context.Context) error { return database.EnsureIndexes(ctx, TokenIndexes()) }

// TokenStore issues and consumes opaque tokens.
type TokenStore struct{}

// NewOpaque returns 32 random bytes, URL-safe base64.
func NewOpaque() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashOpaque(t string) string {
	s := sha256.Sum256([]byte(t))
	return hex.EncodeToString(s[:])
}

// Issue creates and stores a token of the given kind, returning the raw value.
func (TokenStore) Issue(ctx context.Context, kind models.TokenKind, subject bson.ObjectID, ttl time.Duration, ua, ip string) (string, error) {
	raw, err := NewOpaque()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	_, err = tokens().InsertOne(ctx, models.AuthToken{
		Kind: kind, SubjectID: subject, Hash: hashOpaque(raw),
		ExpiresAt: now.Add(ttl), UserAgent: ua, IP: ip, CreatedAt: now,
	})
	return raw, err
}

// Consume atomically marks a live token as used and returns its subject.
// grace allows a token consumed within that window to be accepted again;
// this absorbs concurrent refresh calls (two tabs, StrictMode double effect)
// without logging the user out. Pass 0 for strictly single-use tokens.
func (TokenStore) Consume(ctx context.Context, kind models.TokenKind, raw string, grace time.Duration) (bson.ObjectID, error) {
	now := time.Now().UTC()
	live := bson.A{bson.M{"usedAt": nil}}
	if grace > 0 {
		live = append(live, bson.M{"usedAt": bson.M{"$gt": now.Add(-grace)}})
	}
	var doc models.AuthToken
	err := tokens().FindOneAndUpdate(ctx,
		bson.M{"kind": kind, "hash": hashOpaque(raw), "expiresAt": bson.M{"$gt": now}, "$or": live},
		bson.A{bson.M{"$set": bson.M{"usedAt": bson.M{"$ifNull": bson.A{"$usedAt", now}}}}},
	).Decode(&doc)
	if err != nil {
		if database.IsNoDocuments(err) {
			return bson.ObjectID{}, ErrTokenInvalid
		}
		return bson.ObjectID{}, err
	}
	return doc.SubjectID, nil
}

// Revoke marks a token used and expired without returning it (logout).
// Expiring it too keeps it outside the Consume grace window, which only
// exists for tokens that were legitimately rotated moments ago.
func (TokenStore) Revoke(ctx context.Context, kind models.TokenKind, raw string) error {
	now := time.Now().UTC()
	_, err := tokens().UpdateOne(ctx,
		bson.M{"kind": kind, "hash": hashOpaque(raw)},
		bson.M{"$set": bson.M{"usedAt": now, "expiresAt": now}})
	return err
}

// RevokeAll invalidates every unexpired token of a kind for a subject
// (account disabled, password changed by an admin).
func (TokenStore) RevokeAll(ctx context.Context, kind models.TokenKind, subject bson.ObjectID) error {
	now := time.Now().UTC()
	_, err := tokens().UpdateMany(ctx,
		bson.M{"kind": kind, "userId": subject, "expiresAt": bson.M{"$gt": now}},
		bson.M{"$set": bson.M{"usedAt": now, "expiresAt": now}})
	return err
}
