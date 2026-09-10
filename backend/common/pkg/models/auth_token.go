package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AuthTokensCollection stores opaque refresh/reset tokens (hashed).
const AuthTokensCollection = "auth_tokens"

// TokenKind distinguishes rows in the auth_tokens collection.
type TokenKind string

const (
	TokenRefresh      TokenKind = "refresh"
	TokenReset        TokenKind = "reset"
	TokenAdminRefresh TokenKind = "admin_refresh"
)

// AuthToken is one opaque token row. Hash is sha256(raw); UsedAt marks it
// consumed (rotation) or revoked.
type AuthToken struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	Kind      TokenKind     `bson:"kind"`
	SubjectID bson.ObjectID `bson:"userId"`
	Hash      string        `bson:"hash"`
	ExpiresAt time.Time     `bson:"expiresAt"`
	UsedAt    *time.Time    `bson:"usedAt,omitempty"`
	UserAgent string        `bson:"userAgent,omitempty"`
	IP        string        `bson:"ip,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`
}
