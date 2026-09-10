package config

import (
	"strconv"
	"strings"

	"smartreceipt/backend/common/pkg/middleware"
)

// CORSDefaults are the per-service fallbacks used when system_config has no
// value. Keys: ALLOWED_ORIGINS, CORS_ALLOWED_METHODS, CORS_ALLOWED_HEADERS,
// CORS_EXPOSED_HEADERS (CSV), CORS_ALLOW_CREDENTIALS (bool), CORS_MAX_AGE.
type CORSDefaults struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// LoadCORS resolves the CORS policy: env → system_config → defaults.
func LoadCORS(d CORSDefaults) *middleware.CORSConfig {
	return &middleware.CORSConfig{
		AllowedOrigins:   csv("ALLOWED_ORIGINS", d.AllowedOrigins),
		AllowedMethods:   csv("CORS_ALLOWED_METHODS", d.AllowedMethods),
		AllowedHeaders:   csv("CORS_ALLOWED_HEADERS", d.AllowedHeaders),
		ExposedHeaders:   csv("CORS_EXPOSED_HEADERS", d.ExposedHeaders),
		AllowCredentials: GetConfigBool("CORS_ALLOW_CREDENTIALS", d.AllowCredentials),
		MaxAge:           maxAge("CORS_MAX_AGE", d.MaxAge),
	}
}

func csv(key string, fallback []string) []string {
	raw := GetConfigVal(key, "")
	if raw == "" {
		return fallback
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func maxAge(key string, fallback int) int {
	if v := GetConfigVal(key, ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
