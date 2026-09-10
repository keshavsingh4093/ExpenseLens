// Package config is the admin backend's configuration.
package config

import (
	"errors"
	"time"

	"smartreceipt/backend/common/pkg/config"
)

// DefaultPort is the admin API port.
const DefaultPort = 8091

// Config holds every setting the admin backend needs.
type Config struct {
	config.BaseConfig

	AdminAppURL     string // admin SPA origin (CORS default)
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// UserAPIURL + ConfigReloadSecret let the admin ask the user backend to
	// reload its dynamic config after system_config edits (optional).
	UserAPIURL         string
	ConfigReloadSecret string
}

// AppConfig is the process-wide configuration.
var AppConfig *Config

// GetConfigVal resolves key from env, then system_config, then fallback.
func GetConfigVal(key, fallback string) string { return config.GetConfigVal(key, fallback) }

// Load reads the bootstrap env and builds AppConfig with code defaults.
func Load() error {
	AppConfig = build(config.LoadBase(DefaultPort))
	return nil
}

// Reload rebuilds AppConfig now that dynamic config is loaded.
func Reload() error {
	if AppConfig == nil {
		return Load()
	}
	base := AppConfig.BaseConfig
	AppConfig = build(&base)
	return nil
}

func build(base *config.BaseConfig) *Config {
	return &Config{
		BaseConfig:         *base,
		AdminAppURL:        GetConfigVal("ADMIN_APP_URL", "http://localhost:5174"),
		AccessTokenTTL:     config.GetConfigDuration("ADMIN_ACCESS_TOKEN_TTL", 30*time.Minute),
		RefreshTokenTTL:    config.GetConfigDuration("ADMIN_REFRESH_TOKEN_TTL", 12*time.Hour),
		UserAPIURL:         GetConfigVal("USER_API_URL", "http://localhost:8090"),
		ConfigReloadSecret: GetConfigVal("CONFIG_RELOAD_SECRET", ""),
	}
}

// ValidateProductionSecrets fails fast on placeholder secrets in production.
func ValidateProductionSecrets() error {
	if AppConfig == nil || !AppConfig.IsProd() {
		return nil
	}
	if AppConfig.JWTSecret == "" || AppConfig.JWTSecret == config.DevJWTSecret {
		return errors.New("config: JWT_SECRET must be set to a non-default value when ENV=production")
	}
	return nil
}
