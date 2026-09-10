// Package analytics exposes /api/analytics.
package analytics

import (
	"net/http"
	"strconv"

	gohttp "smartreceipt/backend/common/pkg/http"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/analytics"
)

var svc = analytics.New()

// GetRouteSpecs returns the analytics routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/analytics/summary", Handler: handleSummary},
		{Method: "GET", Path: "/api/analytics/trend", Handler: handleTrend},
		{Method: "GET", Path: "/api/analytics/categories", Handler: handleCategories},
		{Method: "GET", Path: "/api/analytics/merchants", Handler: handleMerchants},
		{Method: "GET", Path: "/api/analytics/stats", Handler: handleStats},
	}
}

func tz(r *http.Request) string {
	if t := r.URL.Query().Get("tz"); t != "" {
		return t
	}
	return "Asia/Kolkata"
}

func rangeFrom(r *http.Request) (analytics.Range, error) {
	q := r.URL.Query()
	rg, err := analytics.ParseRange(q.Get("range"), q.Get("from"), q.Get("to"), analytics.Now(tz(r)))
	if err != nil {
		return rg, gohttp.ErrBadRequest(err.Error())
	}
	return rg, nil
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	out, err := svc.Summary(r.Context(), middleware.UserID(r.Context()), analytics.Now(tz(r)))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, out)
}

func handleTrend(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	g := r.URL.Query().Get("granularity")
	if g == "" {
		g = analytics.DefaultGranularity(rg)
	}
	points, err := svc.ByDay(r.Context(), middleware.UserID(r.Context()), rg)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"range": rg, "granularity": g, "points": analytics.Rollup(points, rg, g)})
}

func handleCategories(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	out, err := svc.ByCategory(r.Context(), middleware.UserID(r.Context()), rg)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"range": rg, "categories": out})
}

func handleMerchants(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	out, err := svc.TopMerchants(r.Context(), middleware.UserID(r.Context()), rg, limit)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"range": rg, "merchants": out})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	rg, err := rangeFrom(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	points, err := svc.ByDay(r.Context(), middleware.UserID(r.Context()), rg)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"range": rg, "stats": analytics.ComputeStats(points, rg)})
}
