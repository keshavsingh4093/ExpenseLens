package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"smartreceipt/backend/common/pkg/middleware"
)

const defaultTimeoutSeconds int64 = 30

// RouteSpec declares one route. Extras are additive middleware applied
// closest to the handler; the foundation (recovery, timeout, auth) cannot be
// bypassed.
type RouteSpec struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
	TimeOut int64 // seconds; 0 = default (30)
	Extras  []func(http.Handler) http.Handler
}

// LoadAPIs registers authenticated routes (EnsureAuth is applied).
func LoadAPIs(specs []RouteSpec, mux chi.Router) { load(specs, mux, true) }

// LoadOpenAPIs registers unauthenticated routes.
func LoadOpenAPIs(specs []RouteSpec, mux chi.Router) { load(specs, mux, false) }

func build(spec RouteSpec, auth bool) http.Handler {
	var h http.Handler = spec.Handler
	for i := len(spec.Extras) - 1; i >= 0; i-- {
		h = spec.Extras[i](h)
	}
	if auth {
		h = middleware.EnsureAuth(h)
	}
	t := defaultTimeoutSeconds
	if spec.TimeOut > 0 {
		t = spec.TimeOut
	}
	h = middleware.WithTimeout(t)(h)
	h = middleware.RecoveryAndClean(h)
	return h
}

func load(specs []RouteSpec, mux chi.Router, auth bool) {
	if mux == nil {
		return
	}
	for _, spec := range specs {
		h := build(spec, auth)
		switch spec.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
			mux.Method(spec.Method, spec.Path, h)
		default:
			panic("gohttp: unsupported method " + spec.Method + " for " + spec.Path)
		}
	}
}

// NotFoundHandler answers unknown routes with the envelope.
func NotFoundHandler(w http.ResponseWriter, _ *http.Request) { NotFound(w, "route not found") }

// MethodNotAllowedHandler answers wrong methods with the envelope.
func MethodNotAllowedHandler(w http.ResponseWriter, _ *http.Request) {
	ErrorWithCode(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}
