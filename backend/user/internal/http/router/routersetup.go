// Package router builds the chi mux for the user backend.
package router

import (
	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	commonmw "smartreceipt/backend/common/pkg/middleware"

	"smartreceipt/backend/user/internal/config"
	"smartreceipt/backend/user/internal/http/api"
	"smartreceipt/backend/user/internal/http/openapi"
	"smartreceipt/backend/user/internal/pkg/middleware"
)

// RouteSpec aliases the common route spec.
type RouteSpec = gohttp.RouteSpec

// InitRouter configures auth, global middleware and every route.
func InitRouter() *chi.Mux {
	r := chi.NewMux()
	commonmw.ConfigureAuth(config.AppConfig.JWTSecret, middleware.NewUserClaims)

	r.Use(commonmw.RequestID)
	r.Use(commonmw.Logging)
	r.NotFound(gohttp.NotFoundHandler)
	r.MethodNotAllowed(gohttp.MethodNotAllowedHandler)

	gohttp.LoadOpenAPIs(openapi.GetRouteSpecs(), r)
	gohttp.LoadAPIs(api.GetRouteSpecs(), r)
	return r
}
