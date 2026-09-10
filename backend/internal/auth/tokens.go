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

	"expenselens/backend/internal/db"
)

// Opaque tokens (refresh, password reset) are random, stored hashed, and
// single-use. This file holds the shared repository for both kinds.

// tokenKind distinguishes rows in the tokens collection.
type tokenKind string

const (
	kindRefresh tokenKind = "refresh"
	kindReset   tokenKind = "reset"
)

type tokenDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Kind      tokenKind     `bson:"kind"`
	UserID    bson.ObjectID `bson:"userId"`
	Hash      string        `bson:"hash"`
	ExpiresAt time.Time     `bson:"expiresAt"`
	UsedAt    *time.Time    `bson:"usedAt,omitempty"`
	UserAgent string        `bson:"userAgent,omitempty"`
	IP        string        `bson:"ip,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// TokenRepo stores opaque tokens.
type TokenRepo struct{ col *mongo.Collection }

// NewTokenRepo binds to the "auth_tokens" collection.
func NewTokenRepo(d *db.DB) *TokenRepo { return &TokenRepo{col: d.Collection("auth_tokens")} }

// Indexes declares token indexes, including a TTL that purges expired rows.
func Indexes() db.IndexSet {
	return db.IndexSet{Collection: "auth_tokens", Models: []mongo.IndexModel{
		{Keys: bson.D{{Key: "hash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "kind", Value: 1}}},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	}}
}

var errTokenInvalid = errors.New("token invalid or expired")

func newOpaque() (string, error) {
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

// issue creates and stores a token of the given kind, returning the raw value.
func (r *TokenRepo) issue(ctx context.Context, kind tokenKind, userID bson.ObjectID, ttl time.Duration, ua, ip string) (string, error) {
	raw, err := newOpaque()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	_, err = r.col.InsertOne(ctx, tokenDoc{
		Kind: kind, UserID: userID, Hash: hashOpaque(raw),
		ExpiresAt: now.Add(ttl), UserAgent: ua, IP: ip, CreatedAt: now,
	})
	return raw, err
}

// consume atomically marks a live token as used and returns its owner.
// grace allows a token that was consumed within that window to be accepted
// again; this absorbs concurrent refresh calls (e.g. two browser tabs or a
// React StrictMode double effect) without logging the user out. Pass 0 for
// strictly single-use tokens.
func (r *TokenRepo) consume(ctx context.Context, kind tokenKind, raw string, grace time.Duration) (bson.ObjectID, error) {
	now := time.Now().UTC()
	live := bson.A{bson.M{"usedAt": nil}}
	if grace > 0 {
		live = append(live, bson.M{"usedAt": bson.M{"$gt": now.Add(-grace)}})
	}
	var doc tokenDoc
	err := r.col.FindOneAndUpdate(ctx,
		bson.M{"kind": kind, "hash": hashOpaque(raw), "expiresAt": bson.M{"$gt": now}, "$or": live},
		bson.A{bson.M{"$set": bson.M{"usedAt": bson.M{"$ifNull": bson.A{"$usedAt", now}}}}},
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return bson.ObjectID{}, errTokenInvalid
		}
		return bson.ObjectID{}, err
	}
	return doc.UserID, nil
}

// revoke marks a token used without returning it (logout).
func (r *TokenRepo) revoke(ctx context.Context, kind tokenKind, raw string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"kind": kind, "hash": hashOpaque(raw)},
		bson.M{"$set": bson.M{"usedAt": time.Now().UTC()}})
	return err
}

// revokeAll invalidates every live token of a kind for a user.
func (r *TokenRepo) revokeAll(ctx context.Context, kind tokenKind, userID bson.ObjectID) error {
	_, err := r.col.UpdateMany(ctx,
		bson.M{"kind": kind, "userId": userID, "usedAt": nil},
		bson.M{"$set": bson.M{"usedAt": time.Now().UTC()}})
	return err
}
