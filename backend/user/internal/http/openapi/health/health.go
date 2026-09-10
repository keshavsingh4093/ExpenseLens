// Package health exposes GET /openapi/health.
package health

import (
	"net/http"

	gohttp "smartreceipt/backend/common/pkg/http"
)

// GetRouteSpecs returns the health routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{{Method: "GET", Path: "/openapi/health", Handler: handleHealth}}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	gohttp.Success(w, map[string]string{"status": "healthy"})
}
