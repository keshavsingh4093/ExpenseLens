// Package reports exposes /api/reports/export (CSV / PDF downloads).
package reports

import (
	"fmt"
	"net/http"

	gohttp "smartreceipt/backend/common/pkg/http"
	usersvc "smartreceipt/backend/common/services/user"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/analytics"
	"smartreceipt/backend/user/internal/services/report"
)

var (
	svc   = report.New()
	users = usersvc.New()
)

// GetRouteSpecs returns the report routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/reports/export", Handler: handleExport, TimeOut: 120},
	}
}

// Attachment writes bytes as a download.
func Attachment(w http.ResponseWriter, name, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tz := q.Get("tz")
	uid := middleware.UserID(r.Context())
	currency, name := "INR", ""
	if u, err := users.FindByID(r.Context(), uid); err == nil {
		name = u.Name
		if u.Preferences.Currency != "" {
			currency = u.Preferences.Currency
		}
		if tz == "" {
			tz = u.Preferences.Timezone
		}
	}
	if tz == "" {
		tz = "Asia/Kolkata"
	}
	rg, err := analytics.ParseRange(q.Get("range"), q.Get("from"), q.Get("to"), analytics.Now(tz))
	if err != nil {
		gohttp.WriteError(w, r, gohttp.ErrBadRequest(err.Error()))
		return
	}
	base := fmt.Sprintf("expenselens-%s-to-%s", rg.From, rg.To)
	switch q.Get("format") {
	case "", "csv":
		body, err := svc.CSV(r.Context(), uid, rg)
		if err != nil {
			gohttp.WriteError(w, r, err)
			return
		}
		Attachment(w, base+".csv", "text/csv; charset=utf-8", body)
	case "pdf":
		body, err := svc.PDF(r.Context(), uid, rg, currency, name)
		if err != nil {
			gohttp.WriteError(w, r, err)
			return
		}
		Attachment(w, base+".pdf", "application/pdf", body)
	default:
		gohttp.WriteError(w, r, gohttp.ErrBadRequest("format must be csv or pdf"))
	}
}
