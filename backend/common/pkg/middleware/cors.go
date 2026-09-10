package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig configures the CORS middleware.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int // seconds
	// AllowOriginFunc is consulted when the origin is not in AllowedOrigins.
	AllowOriginFunc func(origin string) bool
}

// DefaultCORSConfig is permissive and suitable for local development only.
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Requested-With"},
		MaxAge:         86400,
	}
}

func (c *CORSConfig) originAllowed(origin string) (string, bool) {
	for _, o := range c.AllowedOrigins {
		if o == "*" {
			if c.AllowCredentials {
				return origin, true // credentials cannot use the literal "*"
			}
			return "*", true
		}
		if strings.EqualFold(o, origin) {
			return origin, true
		}
	}
	if c.AllowOriginFunc != nil && c.AllowOriginFunc(origin) {
		return origin, true
	}
	return "", false
}

// CORS applies the configured policy and short-circuits preflight requests.
func CORS(cfg *CORSConfig) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultCORSConfig()
	}
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	exposed := strings.Join(cfg.ExposedHeaders, ", ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			allowed, ok := cfg.originAllowed(origin)
			if !ok {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Origin", allowed)
			if cfg.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if exposed != "" {
				h.Set("Access-Control-Expose-Headers", exposed)
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", methods)
				if headers != "" {
					h.Set("Access-Control-Allow-Headers", headers)
				}
				if cfg.MaxAge > 0 {
					h.Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
