// Package router builds the chi mux for the user backend.
package router

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	commonmw "smartreceipt/backend/common/pkg/middleware"

	"smartreceipt/backend/user/internal/config"
	"smartreceipt/backend/user/internal/http/api"
	"smartreceipt/backend/user/internal/http/openapi"
	"smartreceipt/backend/user/internal/pkg/middleware"
	authsvc "smartreceipt/backend/user/internal/services/auth"
)

var auth = authsvc.New()

// activeUserGuard rejects bearer tokens whose subject was disabled or
// deleted after the token was issued.
func activeUserGuard(ctx context.Context, claims jwt.Claims) error {
	c, ok := claims.(*middleware.UserClaims)
	if !ok {
		return gohttp.ErrUnauthorized
	}
	uid, err := bson.ObjectIDFromHex(c.UserID)
	if err != nil {
		return gohttp.ErrUnauthorized
	}
	return auth.EnsureActive(ctx, uid)
}

// RouteSpec aliases the common route spec.
type RouteSpec = gohttp.RouteSpec

// InitRouter configures auth, global middleware and every route.
func InitRouter() *chi.Mux {
	r := chi.NewMux()
	commonmw.ConfigureAuth(config.AppConfig.JWTSecret, middleware.NewUserClaims)
	commonmw.SetAuthGuard(activeUserGuard)

	r.Use(commonmw.RequestID)
	r.Use(commonmw.Logging)
	r.NotFound(gohttp.NotFoundHandler)
	r.MethodNotAllowed(gohttp.MethodNotAllowedHandler)

	gohttp.LoadOpenAPIs(openapi.GetRouteSpecs(), r)
	gohttp.LoadAPIs(api.GetRouteSpecs(), r)
	return r
}
