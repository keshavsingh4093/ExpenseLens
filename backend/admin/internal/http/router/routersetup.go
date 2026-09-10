// Package router builds the chi mux for the admin backend.
package router

import (
	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	commonmw "smartreceipt/backend/common/pkg/middleware"

	"smartreceipt/backend/admin/internal/config"
	"smartreceipt/backend/admin/internal/http/api"
	"smartreceipt/backend/admin/internal/http/openapi"
	"smartreceipt/backend/admin/internal/pkg/middleware"
)

// InitRouter configures auth, global middleware and every route.
func InitRouter() *chi.Mux {
	r := chi.NewMux()
	commonmw.ConfigureAuth(config.AppConfig.JWTSecret, middleware.NewAdminClaims)

	r.Use(commonmw.RequestID)
	r.Use(commonmw.Logging)
	r.NotFound(gohttp.NotFoundHandler)
	r.MethodNotAllowed(gohttp.MethodNotAllowedHandler)

	gohttp.LoadOpenAPIs(openapi.GetRouteSpecs(), r)
	gohttp.LoadAPIs(api.GetRouteSpecs(), r)
	return r
}
