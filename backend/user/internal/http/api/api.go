// Package api aggregates the authenticated routes (/api/...).
package api

import (
	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/user/internal/http/api/analytics"
	"smartreceipt/backend/user/internal/http/api/categories"
	"smartreceipt/backend/user/internal/http/api/receipts"
	"smartreceipt/backend/user/internal/http/api/user"
)

// GetRouteSpecs returns every authenticated route.
func GetRouteSpecs() []gohttp.RouteSpec {
	specs := []gohttp.RouteSpec{}
	specs = append(specs, user.GetRouteSpecs()...)
	specs = append(specs, receipts.GetRouteSpecs()...)
	specs = append(specs, categories.GetRouteSpecs()...)
	specs = append(specs, analytics.GetRouteSpecs()...)
	return specs
}
