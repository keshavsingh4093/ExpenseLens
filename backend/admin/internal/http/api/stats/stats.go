// Package stats exposes GET /api/admin/stats.
package stats

import (
	"net/http"

	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/admin/internal/services/stats"
)

var svc = stats.New()

// GetRouteSpecs returns the route.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{{Method: "GET", Path: "/api/admin/stats", Handler: handleStats}}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	o, err := svc.Overview(r.Context())
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, o)
}
