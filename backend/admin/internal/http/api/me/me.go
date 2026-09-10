// Package me exposes GET /api/admin/me.
package me

import (
	"net/http"

	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/admin/internal/pkg/middleware"
	"smartreceipt/backend/admin/internal/services/adminauth"
)

var svc = adminauth.New()

// GetRouteSpecs returns the route.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{{Method: "GET", Path: "/api/admin/me", Handler: handleMe}}
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	a, err := svc.Me(r.Context(), middleware.AdminID(r.Context()))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, a.ToPublic())
}
