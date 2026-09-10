// Package extraction exposes /api/admin/extraction: stats, failed list, retry.
package extraction

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"

	"smartreceipt/backend/admin/internal/services/audit"
	"smartreceipt/backend/admin/internal/services/extraction"
)

var svc = extraction.New()

// GetRouteSpecs returns the extraction monitoring routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/extraction/stats", Handler: handleStats},
		{Method: "GET", Path: "/api/admin/extraction/failed", Handler: handleFailed},
		{Method: "POST", Path: "/api/admin/extraction/{receiptId}/retry", Handler: handleRetry, TimeOut: 60},
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	out, err := svc.Stats(r.Context(), r.URL.Query().Get("range"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, out)
}

func handleFailed(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	rows, total, err := svc.Failed(r.Context(), page, size)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if size <= 0 || size > 100 {
		size = 20
	}
	if page < 1 {
		page = 1
	}
	gohttp.Success(w, map[string]any{"receipts": rows, "total": total, "page": page, "pageSize": size})
}

func handleRetry(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "receiptId"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	res, err := svc.Retry(r.Context(), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	audit.Log(r, "extraction.retry", audit.TargetReceipt, id.Hex(), nil)
	gohttp.Accepted(w, res)
}
