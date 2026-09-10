// Package middleware holds the HTTP middleware shared by every service:
// bearer-token auth, timeouts, panic recovery, request logging and CORS.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is the type for context keys set by this package.
type ContextKey string

// UserContextKey stores the parsed JWT claims on the request context.
const UserContextKey ContextKey = "claims"

var (
	jwtSecret     []byte
	claimsFactory func() jwt.Claims
)

// ConfigureAuth sets the signing secret and the claims factory used by
// EnsureAuth. The factory returns a pointer to a fresh claims struct; if it
// implements jwt.ClaimsValidator its Validate method runs during parsing.
func ConfigureAuth(secret string, factory func() jwt.Claims) {
	jwtSecret = []byte(secret)
	claimsFactory = factory
}

// ParseToken validates a signed token string into a claims value from the
// configured factory.
func ParseToken(token string) (jwt.Claims, error) {
	if claimsFactory == nil {
		return nil, errors.New("auth not configured")
	}
	claims := claimsFactory()
	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return jwtSecret, nil
	}, jwt.WithExpirationRequired())
	if err != nil || !t.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return t.Claims, nil
}

// EnsureAuth rejects requests without a valid bearer token and stores the
// claims on the context.
func EnsureAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claimsFactory == nil {
			writeJSON(w, http.StatusInternalServerError, "internal", "auth not configured")
			return
		}
		h := r.Header.Get("Authorization")
		if h == "" {
			writeJSON(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		scheme, token, ok := strings.Cut(h, " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			writeJSON(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header")
			return
		}
		claims, err := ParseToken(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
	})
}

// WithClaims attaches claims to ctx under the same key EnsureAuth uses.
func WithClaims(ctx context.Context, claims jwt.Claims) context.Context {
	return context.WithValue(ctx, UserContextKey, claims)
}

// GetClaimsFromContext returns the raw claims stored by EnsureAuth.
func GetClaimsFromContext(ctx context.Context) (jwt.Claims, bool) {
	c, ok := ctx.Value(UserContextKey).(jwt.Claims)
	return c, ok
}
