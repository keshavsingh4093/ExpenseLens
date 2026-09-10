// Package categories exposes /api/admin/categories (system category CRUD).
package categories

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/validate"

	"smartreceipt/backend/admin/internal/services/audit"
	"smartreceipt/backend/admin/internal/services/categories"
)

var svc = categories.New()

// GetRouteSpecs returns the system category routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/admin/categories", Handler: handleList},
		{Method: "POST", Path: "/api/admin/categories", Handler: handleCreate},
		{Method: "PATCH", Path: "/api/admin/categories/{id}", Handler: handleUpdate},
		{Method: "DELETE", Path: "/api/admin/categories/{id}", Handler: handleDelete, TimeOut: 120},
	}
}

type input struct {
	Name  *string `json:"name" validate:"omitempty,min=1,max=40"`
	Slug  *string `json:"slug" validate:"omitempty,min=1,max=40"`
	Icon  *string `json:"icon" validate:"omitempty,max=40"`
	Color *string `json:"color" validate:"omitempty,hexcolor"`
}

func handleList(w http.ResponseWriter, r *http.Request) {
	rows, err := svc.List(r.Context())
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"categories": rows})
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var in input
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if in.Name == nil || strings.TrimSpace(*in.Name) == "" {
		gohttp.ValidationError(w, map[string]string{"name": "is required"})
		return
	}
	slug, icon, color := "", "tag", "#64748b"
	if in.Slug != nil {
		slug = *in.Slug
	}
	if in.Icon != nil {
		icon = *in.Icon
	}
	if in.Color != nil {
		color = *in.Color
	}
	c, err := svc.Create(r.Context(), *in.Name, slug, icon, color)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	audit.Log(r, "category.create", audit.TargetCategory, c.ID.Hex(), map[string]any{"name": c.Name, "slug": c.Slug})
	gohttp.Created(w, c)
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	var in input
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	set := bson.M{}
	if in.Name != nil {
		set["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Slug != nil {
		set["slug"] = models.Slugify(*in.Slug)
	}
	if in.Icon != nil {
		set["icon"] = *in.Icon
	}
	if in.Color != nil {
		set["color"] = *in.Color
	}
	if len(set) == 0 {
		gohttp.BadRequest(w, "nothing to update")
		return
	}
	c, err := svc.Update(r.Context(), id, set)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	audit.Log(r, "category.update", audit.TargetCategory, id.Hex(), map[string]any{"name": c.Name, "slug": c.Slug, "changed": keys(set)})
	gohttp.Success(w, c)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	c, moved, err := svc.Delete(r.Context(), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	audit.Log(r, "category.delete", audit.TargetCategory, id.Hex(), map[string]any{"name": c.Name, "slug": c.Slug, "receiptsMoved": moved})
	gohttp.Success(w, map[string]any{"deleted": true, "receiptsMoved": moved})
}

func keys(m bson.M) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
