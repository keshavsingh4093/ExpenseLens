// Package admins exposes /api/admin/admins: list (any admin) and, for
// superadmins only, create, update (name/role), disable/enable and set
// password.
package admins

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"
	"smartreceipt/backend/common/services/adminuser"

	"smartreceipt/backend/admin/internal/pkg/middleware"
	"smartreceipt/backend/admin/internal/services/admins"
	"smartreceipt/backend/admin/internal/services/audit"
)

var svc = admins.New()

// GetRouteSpecs returns the admin account routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	super := []func(http.Handler) http.Handler{middleware.RequireRole(middleware.RoleSuperadmin)}
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/admins", Handler: handleList},
		{Method: "POST", Path: "/api/admin/admins", Handler: handleCreate, Extras: super},
		{Method: "PATCH", Path: "/api/admin/admins/{id}", Handler: handleUpdate, Extras: super},
		{Method: "POST", Path: "/api/admin/admins/{id}/disable", Handler: handleDisable, Extras: super},
		{Method: "POST", Path: "/api/admin/admins/{id}/enable", Handler: handleEnable, Extras: super},
		{Method: "POST", Path: "/api/admin/admins/{id}/reset-password", Handler: handleResetPassword, Extras: super},
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := adminuser.ListFilter{Query: q.Get("q")}
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
	gohttp.Success(w, map[string]any{"admins": out, "total": total, "page": f.Page, "pageSize": f.PageSize})
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email" validate:"required,email"`
		Name     string `json:"name" validate:"required,min=1,max=80"`
		Password string `json:"password" validate:"required"`
		Role     string `json:"role" validate:"omitempty,oneof=admin superadmin"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	a, err := svc.Create(r.Context(), in.Email, in.Name, in.Password, in.Role)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	audit.Log(r, "admin.create", audit.TargetAdmin, a.ID.Hex(), map[string]any{"email": a.Email, "role": a.Role})
	gohttp.Created(w, map[string]any{"admin": a.ToPublic()})
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	var in struct {
		Name *string `json:"name" validate:"omitempty,min=1,max=80"`
		Role *string `json:"role" validate:"omitempty,oneof=admin superadmin"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if in.Name == nil && in.Role == nil {
		gohttp.WriteError(w, r, gohttp.ErrBadRequest("nothing to update"))
		return
	}
	a, err := svc.Update(r.Context(), middleware.AdminID(r.Context()), id, in.Name, in.Role)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	details := map[string]any{"email": a.Email}
	if in.Name != nil {
		details["name"] = a.Name
	}
	if in.Role != nil {
		details["role"] = a.Role
	}
	audit.Log(r, "admin.update", audit.TargetAdmin, id.Hex(), details)
	gohttp.Success(w, map[string]any{"admin": a.ToPublic()})
}

func setDisabled(w http.ResponseWriter, r *http.Request, disabled bool) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	a, err := svc.SetDisabled(r.Context(), middleware.AdminID(r.Context()), id, disabled)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	action := "admin.enable"
	if disabled {
		action = "admin.disable"
	}
	audit.Log(r, action, audit.TargetAdmin, id.Hex(), map[string]any{"email": a.Email})
	gohttp.Success(w, map[string]any{"admin": a.ToPublic()})
}

func handleDisable(w http.ResponseWriter, r *http.Request) { setDisabled(w, r, true) }
func handleEnable(w http.ResponseWriter, r *http.Request)  { setDisabled(w, r, false) }

func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	var in struct {
		Password string `json:"password" validate:"required"`
	}
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	a, err := svc.ResetPassword(r.Context(), id, in.Password)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	// Never log the password.
	audit.Log(r, "admin.reset_password", audit.TargetAdmin, id.Hex(), map[string]any{"email": a.Email})
	gohttp.Success(w, map[string]any{"admin": a.ToPublic()})
}
