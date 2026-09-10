// Package categories exposes /api/categories.
package categories

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/models"
	"smartreceipt/backend/common/pkg/validate"
	"smartreceipt/backend/common/services/category"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/receipt"
)

var (
	svc      = category.New()
	receipts = receipt.New()
)

// GetRouteSpecs returns the category routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/categories", Handler: handleList},
		{Method: "POST", Path: "/api/categories", Handler: handleCreate},
		{Method: "PATCH", Path: "/api/categories/{id}", Handler: handleUpdate},
		{Method: "DELETE", Path: "/api/categories/{id}", Handler: handleDelete},
	}
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, category.ErrNotFound):
		return gohttp.ErrNotFound
	case errors.Is(err, category.ErrExists):
		return gohttp.ErrConflict("category_exists", "a category with that name already exists")
	case errors.Is(err, category.ErrSystem):
		return gohttp.ErrForbidden
	}
	return err
}

func handleList(w http.ResponseWriter, r *http.Request) {
	out, err := svc.ListForUser(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"categories": out})
}

type categoryInput struct {
	Name  *string `json:"name" validate:"omitempty,min=1,max=40"`
	Icon  *string `json:"icon" validate:"omitempty,max=40"`
	Color *string `json:"color" validate:"omitempty,hexcolor"`
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var in categoryInput
	if err := gohttp.Decode(r, &in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := validate.Struct(in); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if in.Name == nil {
		gohttp.ValidationError(w, map[string]string{"name": "is required"})
		return
	}
	icon, color := "tag", "#64748b"
	if in.Icon != nil {
		icon = *in.Icon
	}
	if in.Color != nil {
		color = *in.Color
	}
	c, err := svc.Create(r.Context(), middleware.UserID(r.Context()), *in.Name, icon, color)
	if err != nil {
		gohttp.WriteError(w, r, mapErr(err))
		return
	}
	gohttp.Created(w, c)
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	var in categoryInput
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
		set["name"] = *in.Name
		set["slug"] = models.Slugify(*in.Name)
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
	uid := middleware.UserID(r.Context())
	if err := svc.Update(r.Context(), uid, id, set); err != nil {
		gohttp.WriteError(w, r, mapErr(err))
		return
	}
	c, err := svc.Get(r.Context(), uid, id)
	if err != nil {
		gohttp.WriteError(w, r, mapErr(err))
		return
	}
	gohttp.Success(w, c)
}

// handleDelete removes a user category and moves its receipts to "others".
func handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	uid := middleware.UserID(r.Context())
	c, err := svc.Get(r.Context(), uid, id)
	if err != nil {
		gohttp.WriteError(w, r, mapErr(err))
		return
	}
	if c.IsSystem {
		gohttp.WriteError(w, r, gohttp.ErrForbidden)
		return
	}
	others, err := svc.BySlug(r.Context(), uid, models.OthersSlug)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if _, err := receipts.ReassignCategory(r.Context(), uid, id, others.ID); err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	if err := svc.Delete(r.Context(), uid, id); err != nil {
		gohttp.WriteError(w, r, mapErr(err))
		return
	}
	gohttp.NoContent(w)
}
