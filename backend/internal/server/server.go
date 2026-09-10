// Package server wires the HTTP router, middleware, and response helpers.
package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"expenselens/backend/internal/config"
)

// Router is the top-level chi router. Domain packages mount onto API().
type Router struct {
	*chi.Mux
	api chi.Router
}

// New builds the router with global middleware and the /api/v1 group.
func New(cfg *config.Config) *Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AppURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	var api chi.Router
	r.Route("/api/v1", func(sub chi.Router) {
		api = sub
		sub.NotFound(func(w http.ResponseWriter, r *http.Request) { Error(w, r, ErrNotFound) })
	})
	return &Router{Mux: r, api: api}
}

// API returns the /api/v1 sub-router for domain packages to mount routes on.
func (r *Router) API() chi.Router { return r.api }
