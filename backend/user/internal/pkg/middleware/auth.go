// Package middleware holds the user portal's JWT claims and context helpers.
// The bearer-token check itself is common/pkg/middleware.EnsureAuth.
package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	commonmw "smartreceipt/backend/common/pkg/middleware"
	"smartreceipt/backend/common/pkg/models"
)

// ClaimTypeUser marks access tokens issued to end users.
const ClaimTypeUser = "user"

// UserClaims is the access-token payload for the user portal.
type UserClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

// Validate rejects tokens minted for another portal (admin) and tokens
// without a subject. Called automatically by jwt.ParseWithClaims.
func (c *UserClaims) Validate() error {
	if c.Type != ClaimTypeUser || c.UserID == "" {
		return errors.New("not a user token")
	}
	return nil
}

// NewUserClaims is the claims factory for commonmw.ConfigureAuth.
func NewUserClaims() jwt.Claims { return &UserClaims{} }

// ClaimsFor builds the claims for a user session (auth.ClaimsFactory).
func ClaimsFor(u *models.User, issuedAt, expiresAt time.Time) jwt.Claims {
	return &UserClaims{
		UserID: u.ID.Hex(), Email: u.Email, Name: u.Name, Type: ClaimTypeUser,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.Hex(),
			Issuer:    "expenselens",
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
}

// GetUserFromContext returns the typed claims stored by EnsureAuth.
func GetUserFromContext(ctx context.Context) (*UserClaims, bool) {
	c, ok := ctx.Value(commonmw.UserContextKey).(*UserClaims)
	return c, ok
}

// UserID returns the authenticated user's id, or the zero id when absent.
func UserID(ctx context.Context) bson.ObjectID {
	c, ok := GetUserFromContext(ctx)
	if !ok {
		return bson.ObjectID{}
	}
	id, _ := bson.ObjectIDFromHex(c.UserID)
	return id
}
