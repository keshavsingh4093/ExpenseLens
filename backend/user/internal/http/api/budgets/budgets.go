// Package budgets exposes /api/budgets.
package budgets

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/analytics"
	"smartreceipt/backend/user/internal/services/budget"
)

var svc = budget.New()

// GetRouteSpecs returns the budget routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/budgets", Handler: handleList},
		{Method: "PUT", Path: "/api/budgets", Handler: handleUpsert},
		{Method: "DELETE", Path: "/api/budgets/{id}", Handler: handleDelete},
	}
}

func monthFrom(r *http.Request) string {
	if m := r.URL.Query().Get("month"); m != "" {
		return m
	}
	tz := r.URL.Query().Get("tz")
	if tz == "" {
		tz = "Asia/Kolkata"
	}
	return analytics.Now(tz).Format("2006-01")
}

func handleList(w http.ResponseWriter, r *http.Request) {
	month := monthFrom(r)
	rows, sum, err := svc.List(r.Context(), middleware.UserID(r.Context()), month)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"month": month, "budgets": rows, "summary": sum})
}

func handleUpsert(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CategoryID string `json:"categoryId" validate:"required"`
		Month      string `json:"month" validate:"required"`
		Limit      int64  `json:"limit" validate:"required"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	row, err := svc.Upsert(r.Context(), middleware.UserID(r.Context()), in.CategoryID, in.Month, in.Limit)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, row)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := svc.Delete(r.Context(), middleware.UserID(r.Context()), id); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.NoContent(w)
}
