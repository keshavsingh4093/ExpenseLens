// Package router builds the chi mux for the admin backend.
package router

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	commonmw "smartreceipt/backend/common/pkg/middleware"

	"smartreceipt/backend/admin/internal/config"
	"smartreceipt/backend/admin/internal/http/api"
	"smartreceipt/backend/admin/internal/http/openapi"
	"smartreceipt/backend/admin/internal/pkg/middleware"
	"smartreceipt/backend/admin/internal/services/adminauth"
)

var auth = adminauth.New()

// activeAdminGuard rejects bearer tokens whose admin was disabled or deleted
// after the token was issued (403 account_disabled / 401).
func activeAdminGuard(ctx context.Context, claims jwt.Claims) error {
	c, ok := claims.(*middleware.AdminClaims)
	if !ok {
		return gohttp.ErrUnauthorized
	}
	id, err := bson.ObjectIDFromHex(c.AdminID)
	if err != nil {
		return gohttp.ErrUnauthorized
	}
	return auth.EnsureActive(ctx, id)
}

// InitRouter configures auth, global middleware and every route.
func InitRouter() *chi.Mux {
	r := chi.NewMux()
	commonmw.ConfigureAuth(config.AppConfig.JWTSecret, middleware.NewAdminClaims)
	commonmw.SetAuthGuard(activeAdminGuard)

	r.Use(commonmw.RequestID)
	r.Use(commonmw.Logging)
	r.NotFound(gohttp.NotFoundHandler)
	r.MethodNotAllowed(gohttp.MethodNotAllowedHandler)

	gohttp.LoadOpenAPIs(openapi.GetRouteSpecs(), r)
	gohttp.LoadAPIs(api.GetRouteSpecs(), r)
	return r
}
