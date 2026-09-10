package categories

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"expenselens/backend/internal/server"
)

// Reassigner moves receipts off a category being deleted (implemented by receipts).
type Reassigner interface {
	ReassignCategory(ctx context.Context, uid, from, to bson.ObjectID) (int64, error)
}

// Handler exposes /categories.
type Handler struct {
	repo   *Repo
	userID func(context.Context) bson.ObjectID
	reas   Reassigner
}

// NewHandler constructs the handler.
func NewHandler(repo *Repo, userID func(context.Context) bson.ObjectID, reas Reassigner) *Handler {
	return &Handler{repo: repo, userID: userID, reas: reas}
}

// Mount registers routes (caller applies auth middleware).
func (h *Handler) Mount(r chi.Router) {
	r.Route("/categories", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return server.ErrNotFound
	case errors.Is(err, ErrExists):
		return server.Conflict("category_exists", "a category with that name already exists")
	case errors.Is(err, ErrSystem):
		return server.ErrForbidden
	}
	return err
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	out, err := h.repo.ListForUser(r.Context(), h.userID(r.Context()))
	if err != nil {
		server.Error(w, r, err)
		return
	}
	server.JSON(w, http.StatusOK, map[string]any{"categories": out})
}

type categoryInput struct {
	Name  *string `json:"name" validate:"omitempty,min=1,max=40"`
	Icon  *string `json:"icon" validate:"omitempty,max=40"`
	Color *string `json:"color" validate:"omitempty,hexcolor"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in categoryInput
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	if in.Name == nil {
		server.Error(w, r, server.Validation(map[string]string{"name": "is required"}))
		return
	}
	icon, color := "tag", "#64748b"
	if in.Icon != nil {
		icon = *in.Icon
	}
	if in.Color != nil {
		color = *in.Color
	}
	c, err := h.repo.Create(r.Context(), h.userID(r.Context()), *in.Name, icon, color)
	if err != nil {
		server.Error(w, r, mapErr(err))
		return
	}
	server.JSON(w, http.StatusCreated, c)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		server.Error(w, r, server.ErrNotFound)
		return
	}
	var in categoryInput
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	set := bson.M{}
	if in.Name != nil {
		set["name"] = *in.Name
		set["slug"] = Slugify(*in.Name)
	}
	if in.Icon != nil {
		set["icon"] = *in.Icon
	}
	if in.Color != nil {
		set["color"] = *in.Color
	}
	if len(set) == 0 {
		server.Error(w, r, server.BadRequest("nothing to update"))
		return
	}
	uid := h.userID(r.Context())
	if err := h.repo.Update(r.Context(), uid, id, set); err != nil {
		server.Error(w, r, mapErr(err))
		return
	}
	c, err := h.repo.Get(r.Context(), uid, id)
	if err != nil {
		server.Error(w, r, mapErr(err))
		return
	}
	server.JSON(w, http.StatusOK, c)
}

// delete removes a user category and moves its receipts to "others".
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := bson.ObjectIDFromHex(chi.URLParam(r, "id"))
	if err != nil {
		server.Error(w, r, server.ErrNotFound)
		return
	}
	uid := h.userID(r.Context())
	c, err := h.repo.Get(r.Context(), uid, id)
	if err != nil {
		server.Error(w, r, mapErr(err))
		return
	}
	if c.IsSystem {
		server.Error(w, r, server.ErrForbidden)
		return
	}
	others, err := h.repo.BySlug(r.Context(), uid, OthersSlug)
	if err != nil {
		server.Error(w, r, err)
		return
	}
	if h.reas != nil {
		if _, err := h.reas.ReassignCategory(r.Context(), uid, id, others.ID); err != nil {
			server.Error(w, r, err)
			return
		}
	}
	if err := h.repo.Delete(r.Context(), uid, id); err != nil {
		server.Error(w, r, mapErr(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
