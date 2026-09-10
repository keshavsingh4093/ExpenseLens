// Package warranties exposes /api/warranties.
package warranties

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/warranty"
)

var svc = warranty.New()

// GetRouteSpecs returns the warranty routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/warranties", Handler: handleList},
		{Method: "POST", Path: "/api/warranties", Handler: handleCreate},
		{Method: "GET", Path: "/api/warranties/{id}", Handler: handleGet},
		{Method: "PATCH", Path: "/api/warranties/{id}", Handler: handleUpdate},
		{Method: "DELETE", Path: "/api/warranties/{id}", Handler: handleDelete},
	}
}

func decode(r *http.Request) (warranty.Input, error) {
	var in warranty.Input
	if err := gohttp.Decode(r, &in); err != nil {
		return in, err
	}
	// An empty expiry means "derive from duration", not an invalid date.
	if in.ExpiryDate != nil && *in.ExpiryDate == "" {
		in.ExpiryDate = nil
	}
	if err := validate.Struct(in); err != nil {
		return in, err
	}
	return in, nil
}

func handleList(w http.ResponseWriter, r *http.Request) {
	rows, counts, err := svc.List(r.Context(), middleware.UserID(r.Context()), r.URL.Query().Get("status"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"warranties": rows, "counts": counts})
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	in, err := decode(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	out, err := svc.Create(r.Context(), middleware.UserID(r.Context()), in)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Created(w, out)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	out, err := svc.Get(r.Context(), middleware.UserID(r.Context()), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, out)
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	in, err := decode(r)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	out, err := svc.Update(r.Context(), middleware.UserID(r.Context()), id, in)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, out)
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
