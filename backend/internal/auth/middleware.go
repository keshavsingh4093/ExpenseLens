package auth

import (
	"context"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"expenselens/backend/internal/server"
)

type ctxKey struct{}

// RequireAuth rejects requests without a valid bearer access token and stores
// the user id in the request context.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				server.Error(w, r, server.ErrUnauthorized)
				return
			}
			sub, err := ParseAccessToken(secret, strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				server.Error(w, r, server.ErrUnauthorized)
				return
			}
			uid, err := bson.ObjectIDFromHex(sub)
			if err != nil {
				server.Error(w, r, server.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, uid)))
		})
	}
}

// UserID returns the authenticated user id from the context.
func UserID(ctx context.Context) bson.ObjectID {
	id, _ := ctx.Value(ctxKey{}).(bson.ObjectID)
	return id
}
