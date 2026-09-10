// Package openapi aggregates the unauthenticated routes (/openapi/...).
package openapi

import (
	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/user/internal/http/openapi/auth"
	"smartreceipt/backend/user/internal/http/openapi/configreload"
	"smartreceipt/backend/user/internal/http/openapi/files"
	"smartreceipt/backend/user/internal/http/openapi/health"
)

// GetRouteSpecs returns every open route.
func GetRouteSpecs() []gohttp.RouteSpec {
	specs := []gohttp.RouteSpec{}
	specs = append(specs, health.GetRouteSpecs()...)
	specs = append(specs, auth.GetRouteSpecs()...)
	specs = append(specs, files.GetRouteSpecs()...)
	specs = append(specs, configreload.GetRouteSpecs()...)
	return specs
}
