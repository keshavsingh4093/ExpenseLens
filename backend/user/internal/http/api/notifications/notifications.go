// Package notifications exposes /api/notifications.
package notifications

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/validate"

	"smartreceipt/backend/user/internal/pkg/middleware"
	"smartreceipt/backend/user/internal/services/notification"
)

var svc = notification.New()

// GetRouteSpecs returns the notification routes.
func GetRouteSpecs() []gohttp.RouteSpec {
	return []gohttp.RouteSpec{
		{Method: "GET", Path: "/api/notifications", Handler: handleList},
		{Method: "GET", Path: "/api/notifications/unread-count", Handler: handleUnread},
		{Method: "POST", Path: "/api/notifications/read-all", Handler: handleReadAll},
		{Method: "POST", Path: "/api/notifications/{id}/read", Handler: handleRead},
	}
}

func handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("pageSize"))
	out, err := svc.List(r.Context(), middleware.UserID(r.Context()), page, size)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, out)
}

func handleUnread(w http.ResponseWriter, r *http.Request) {
	n, err := svc.UnreadCount(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"unread": n})
}

func handleRead(w http.ResponseWriter, r *http.Request) {
	id, err := validate.ObjectID(chi.URLParam(r, "id"))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	n, err := svc.Read(r.Context(), middleware.UserID(r.Context()), id)
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, n)
}

func handleReadAll(w http.ResponseWriter, r *http.Request) {
	n, err := svc.ReadAll(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		gohttp.WriteError(w, r, err)
		return
	}
	gohttp.Success(w, map[string]any{"updated": n})
}
