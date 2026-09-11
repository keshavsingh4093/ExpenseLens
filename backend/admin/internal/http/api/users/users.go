// Package users exposes /api/admin/users: list/search, get, disable/enable,
// delete (superadmin) and password-reset links.
package users

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"
	usersvc "smartreceipt/backend/common/services/user"

	"smartreceipt/backend/admin/internal/pkg/middleware"
	"smartreceipt/backend/admin/internal/services/audit"
	"smartreceipt/backend/admin/internal/services/users"
)

var svc = users.New()

// GetRouteSpecs returns the user management routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/users", Handler: handleList},
		{Method: "GET", Path: "/api/admin/users/{id}", Handler: handleGet},
		{Method: "POST", Path: "/api/admin/users/{id}/disable", Handler: handleDisable},
		{Method: "POST", Path: "/api/admin/users/{id}/enable", Handler: handleEnable},
		{Method: "POST", Path: "/api/admin/users/{id}/reset-password-link", Handler: handleResetLink},
		{Method: "DELETE", Path: "/api/admin/users/{id}", Handler: handleDelete, TimeOut: 120,
			Extras: []func(http.Handler) http.Handler{middleware.RequireRole(middleware.RoleSuperadmin)}},
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := usersvc.ListFilter{Query: q.Get("q"), Status: q.Get("status")}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	rows, total, err := svc.List(r.Context(), f)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	out := make([]any, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToPublic())
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.Page < 1 {
		f.Page = 1
	}
	gohttp.Success(w, map[string]any{"users": out, "total": total, "page": f.Page, "pageSize": f.PageSize})
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	u, n, err := svc.Get(r.Context(), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"user": u.ToPublic(), "receiptCount": n})
}

func setDisabled(w http.ResponseWriter, r *http.Request, disabled bool) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	u, err := svc.SetDisabled(r.Context(), id, disabled)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	action := "user.enable"
	if disabled {
		action = "user.disable"
	}
	audit.Log(r, action, audit.TargetUser, id.Hex(), map[string]any{"email": u.Email})
	gohttp.Success(w, map[string]any{"user": u.ToPublic()})
}

func handleDisable(w http.ResponseWriter, r *http.Request) { setDisabled(w, r, true) }
func handleEnable(w http.ResponseWriter, r *http.Request)  { setDisabled(w, r, false) }

// handleResetLink issues a password-reset link. The body is optional:
// {"send": true} emails the link to the user instead of returning it.
func handleResetLink(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	var in struct {
		Send bool `json:"send"`
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if len(bytes.TrimSpace(raw)) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(raw))
		if err := gohttp.Decode(r, &in); err != nil {
			gohttp.WriteError(w, r, err)
			return
		}
	}
	if in.Send {
		exp, err := svc.EmailResetLink(r.Context(), id)
		if err != nil {
			gohttp.WriteError(w, r, err)
			return
		}
		audit.Log(r, "user.reset_link", audit.TargetUser, id.Hex(), map[string]any{"emailed": true})
		gohttp.Success(w, map[string]any{"sent": true, "expiresAt": exp})
		return
	}
	link, exp, err := svc.ResetLink(r.Context(), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	// The link itself is a credential: never write it to the audit log.
	audit.Log(r, "user.reset_link", audit.TargetUser, id.Hex(), map[string]any{"emailed": false})
	gohttp.Success(w, map[string]any{"link": link, "expiresAt": exp})
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	email, err := svc.Delete(r.Context(), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	audit.Log(r, "user.delete", audit.TargetUser, id.Hex(), map[string]any{"email": email})
	gohttp.NoContent(w)
}
