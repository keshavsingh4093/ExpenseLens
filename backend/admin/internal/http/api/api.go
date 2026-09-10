// Package api aggregates the admin backend's authenticated routes.
package api

import (
	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/admin/internal/http/api/me"
	"smartreceipt/backend/admin/internal/http/api/stats"
	"smartreceipt/backend/admin/internal/http/api/systemconfig"
	"smartreceipt/backend/admin/internal/http/api/users"
)

// GetRouteSpecs returns every authenticated route.
func GetRouteSpecs() []gohttp.RouteSpec {
	specs := []gohttp.RouteSpec{}
	specs = append(specs, me.GetRouteSpecs()...)
	specs = append(specs, users.GetRouteSpecs()...)
	specs = append(specs, stats.GetRouteSpecs()...)
	specs = append(specs, systemconfig.GetRouteSpecs()...)
	return specs
}
