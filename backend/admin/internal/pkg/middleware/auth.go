// Package middleware holds the admin portal's JWT claims and context helpers.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	commonmw "smartreceipt/backend/common/pkg/middleware"
	"smartreceipt/backend/common/pkg/models"
)

// ClaimTypeAdmin marks access tokens issued to admins.
const ClaimTypeAdmin = "admin"

// AdminClaims is the access-token payload for the admin portal.
type AdminClaims struct {
	AdminID string `json:"adminId"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Type    string `json:"type"`
	jwt.RegisteredClaims
}

// Validate rejects tokens minted for the user portal.
func (c *AdminClaims) Validate() error {
	if c.Type != ClaimTypeAdmin || c.AdminID == "" {
		return errors.New("not an admin token")
	}
	return nil
}

// NewAdminClaims is the claims factory for commonmw.ConfigureAuth.
func NewAdminClaims() jwt.Claims { return &AdminClaims{} }

// ClaimsFor builds the claims for an admin session.
func ClaimsFor(a *models.AdminUser, issuedAt, expiresAt time.Time) jwt.Claims {
	return &AdminClaims{
		AdminID: a.ID.Hex(), Email: a.Email, Name: a.Name, Role: a.Role, Type: ClaimTypeAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   a.ID.Hex(),
			Issuer:    "expenselens-admin",
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
}

// GetAdminFromContext returns the typed claims stored by EnsureAuth.
func GetAdminFromContext(ctx context.Context) (*AdminClaims, bool) {
	c, ok := ctx.Value(commonmw.UserContextKey).(*AdminClaims)
	return c, ok
}

// AdminID returns the authenticated admin's id, or the zero id when absent.
func AdminID(ctx context.Context) bson.ObjectID {
	c, ok := GetAdminFromContext(ctx)
	if !ok {
		return bson.ObjectID{}
	}
	id, _ := bson.ObjectIDFromHex(c.AdminID)
	return id
}

// RoleSuperadmin can do everything, including destructive user actions.
const RoleSuperadmin = "superadmin"

// RequireRole is a RouteSpec extra that rejects admins without the role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := GetAdminFromContext(r.Context())
			if !ok || c.Role != role {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "requires the " + role + " role", "code": "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
