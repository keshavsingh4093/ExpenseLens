// Package users exposes /api/admin/users (list/search, get).
package users

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"
	"smartreceipt/backend/common/services/receipt"
	usersvc "smartreceipt/backend/common/services/user"
)

var (
	users    = usersvc.New()
	receipts = receipt.New()
)

// GetRouteSpecs returns the user management routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/users", Handler: handleList},
		{Method: "GET", Path: "/api/admin/users/{id}", Handler: handleGet},
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := usersvc.ListFilter{Query: q.Get("q")}
	f.Page, _ = strconv.Atoi(q.Get("page"))
	f.PageSize, _ = strconv.Atoi(q.Get("pageSize"))
	rows, total, err := users.List(r.Context(), f)
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
	u, err := users.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, usersvc.ErrNotFound) {
			err = gohttp.ErrNotFound
		}
		gohttp.WriteError(w, r, err)
		return
	}
	n, _ := receipts.Count(r.Context(), bson.M{"userId": id})
	gohttp.Success(w, map[string]any{"user": u.ToPublic(), "receiptCount": n})
}
