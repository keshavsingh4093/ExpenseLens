// Package api aggregates the authenticated routes (/api/...).
package api

import (
	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/user/internal/http/api/analytics"
	"smartreceipt/backend/user/internal/http/api/budgets"
	"smartreceipt/backend/user/internal/http/api/categories"
	"smartreceipt/backend/user/internal/http/api/notifications"
	"smartreceipt/backend/user/internal/http/api/receipts"
	"smartreceipt/backend/user/internal/http/api/reports"
	"smartreceipt/backend/user/internal/http/api/user"
	"smartreceipt/backend/user/internal/http/api/warranties"
)

// GetRouteSpecs returns every authenticated route.
func GetRouteSpecs() []gohttp.RouteSpec {
	specs := []gohttp.RouteSpec{}
	specs = append(specs, user.GetRouteSpecs()...)
	specs = append(specs, receipts.GetRouteSpecs()...)
	specs = append(specs, categories.GetRouteSpecs()...)
	specs = append(specs, analytics.GetRouteSpecs()...)
	specs = append(specs, budgets.GetRouteSpecs()...)
	specs = append(specs, warranties.GetRouteSpecs()...)
	specs = append(specs, notifications.GetRouteSpecs()...)
	specs = append(specs, reports.GetRouteSpecs()...)
	return specs
}
