// Package audit exposes GET /api/admin/audit.
package audit

import (
	"net/http"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/services/adminaudit"

	auditsvc "smartreceipt/backend/admin/internal/services/audit"
)

// GetRouteSpecs returns the audit log routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/audit", Handler: handleList},
		{Method: "GET", Path: "/api/admin/audit/actions", Handler: handleActions},
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := adminaudit.ListFilter{Action: q.Get("action"), TargetID: q.Get("targetId")}
	if s := q.Get("adminId"); s != "" {
		id, err := bson.ObjectIDFromHex(s)
		if err != nil {
			gohttp.ValidationError(w, map[string]string{"adminId": "is invalid"})
			return
		}
		f.AdminID = &id
	}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	rows, total, err := auditsvc.List(r.Context(), f)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	f.Normalize()
	gohttp.Success(w, map[string]any{"entries": rows, "total": total, "page": f.Page, "pageSize": f.PageSize})
}

func handleActions(w http.ResponseWriter, r *http.Request) {
	out, err := auditsvc.Actions(r.Context())
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"actions": out})
}
