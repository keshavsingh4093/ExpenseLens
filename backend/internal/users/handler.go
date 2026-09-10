package users

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/v2/bson"

	"expenselens/backend/internal/server"
)

// Deleter cascades an account deletion (implemented by the auth service).
type Deleter interface {
	DeleteAccount(ctx context.Context, uid bson.ObjectID) error
}

// Handler exposes /me routes. userID extracts the caller from context
// (provided by the auth middleware; injected to avoid an import cycle).
type Handler struct {
	repo    *Repo
	userID  func(context.Context) bson.ObjectID
	deleter Deleter
}

// NewHandler constructs the handler.
func NewHandler(repo *Repo, userID func(context.Context) bson.ObjectID, d Deleter) *Handler {
	return &Handler{repo: repo, userID: userID, deleter: d}
}

// Mount registers routes; the caller wraps them with the auth middleware.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/me", func(r chi.Router) {
		r.Get("/", h.get)
		r.Patch("/", h.patch)
		r.Patch("/preferences", h.patchPreferences)
		r.Delete("/", h.delete)
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	u, err := h.repo.FindByID(r.Context(), h.userID(r.Context()))
	if err != nil {
		server.Error(w, r, server.ErrUnauthorized)
		return
	}
	server.JSON(w, http.StatusOK, u.ToPublic())
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      *string `json:"name" validate:"omitempty,min=1,max=80"`
		AvatarURL *string `json:"avatarUrl" validate:"omitempty,url,max=2048"`
	}
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
		set["name"] = strings.TrimSpace(*in.Name)
	}
	if in.AvatarURL != nil {
		set["avatarUrl"] = *in.AvatarURL
	}
	if len(set) == 0 {
		server.Error(w, r, server.BadRequest("nothing to update"))
		return
	}
	uid := h.userID(r.Context())
	if err := h.repo.Update(r.Context(), uid, set); err != nil {
		server.Error(w, r, err)
		return
	}
	h.get(w, r)
}

func (h *Handler) patchPreferences(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Currency      *string `json:"currency" validate:"omitempty,len=3,alpha"`
		Language      *string `json:"language" validate:"omitempty,min=2,max=8"`
		Theme         *string `json:"theme" validate:"omitempty,oneof=light dark system"`
		Timezone      *string `json:"timezone" validate:"omitempty,min=1,max=64"`
		Notifications *struct {
			BudgetAlerts      *bool `json:"budgetAlerts"`
			WarrantyReminders *bool `json:"warrantyReminders"`
			WeeklySummary     *bool `json:"weeklySummary"`
		} `json:"notifications"`
	}
	if err := server.Decode(r, &in); err != nil {
		server.Error(w, r, err)
		return
	}
	if err := server.Validate(in); err != nil {
		server.Error(w, r, err)
		return
	}
	set := bson.M{}
	if in.Currency != nil {
		set["preferences.currency"] = strings.ToUpper(*in.Currency)
	}
	if in.Language != nil {
		set["preferences.language"] = *in.Language
	}
	if in.Theme != nil {
		set["preferences.theme"] = *in.Theme
	}
	if in.Timezone != nil {
		set["preferences.timezone"] = *in.Timezone
	}
	if n := in.Notifications; n != nil {
		if n.BudgetAlerts != nil {
			set["preferences.notifications.budgetAlerts"] = *n.BudgetAlerts
		}
		if n.WarrantyReminders != nil {
			set["preferences.notifications.warrantyReminders"] = *n.WarrantyReminders
		}
		if n.WeeklySummary != nil {
			set["preferences.notifications.weeklySummary"] = *n.WeeklySummary
		}
	}
	if len(set) == 0 {
		server.Error(w, r, server.BadRequest("nothing to update"))
		return
	}
	if err := h.repo.Update(r.Context(), h.userID(r.Context()), set); err != nil {
		server.Error(w, r, err)
		return
	}
	h.get(w, r)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.deleter.DeleteAccount(r.Context(), h.userID(r.Context())); err != nil {
		server.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
