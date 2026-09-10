// Package openapi aggregates the admin backend's unauthenticated routes.
package openapi

import (
	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/admin/internal/http/openapi/auth"
	"smartreceipt/backend/admin/internal/http/openapi/health"
)

// GetRouteSpecs returns every open route.
func GetRouteSpecs() []gohttp.RouteSpec {
	specs := []gohttp.RouteSpec{}
	specs = append(specs, health.GetRouteSpecs()...)
	specs = append(specs, auth.GetRouteSpecs()...)
	return specs
}
