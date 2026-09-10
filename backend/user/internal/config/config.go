// Package config is the user backend's configuration: the common BaseConfig
// (bootstrap .env) plus every other key resolved through GetConfigVal
// (env → system_config → default).
package config

import (
	"errors"
	"time"

	"smartreceipt/backend/common/pkg/config"
	"smartreceipt/backend/common/pkg/extraction"
	"smartreceipt/backend/common/pkg/mail"
	"smartreceipt/backend/common/services/auth"
)

// DefaultPort is the user API port (8080 is taken on the dev machine).
const DefaultPort = 8090

// Config holds every setting the user backend needs.
type Config struct {
	config.BaseConfig

	AppURL string // SPA origin (CORS, redirects, emailed links)
	APIURL string // this service's public URL (OAuth redirect URIs, local file URLs)

	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Storage: local | cloudinary
	StorageProvider string
	LocalStorageDir string
	CloudinaryURL   string
	MaxUploadMB     int64

	// Extraction: gemini | fake
	Extractor         string
	GeminiAPIKey      string
	GeminiModel       string
	ExtractionWorkers int

	OAuth auth.OAuthConfig
	Mail  mail.Config

	// ConfigReloadSecret protects POST /openapi/config/reload (empty = disabled).
	ConfigReloadSecret string
}

// AppConfig is the process-wide configuration.
var AppConfig *Config

// GetConfigVal resolves key from env, then system_config, then fallback.
func GetConfigVal(key, fallback string) string { return config.GetConfigVal(key, fallback) }

// Load reads the bootstrap env and builds AppConfig with code defaults.
// Dynamic (DB) values are not available yet; call Reload after
// InitDynamicConfig.
func Load() error {
	base := config.LoadBase(DefaultPort)
	AppConfig = build(base)
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
		BaseConfig: *base,
		AppURL:     GetConfigVal("APP_URL", "http://localhost:5173"),
		APIURL:     GetConfigVal("API_URL", "http://localhost:8090"),

		AccessTokenTTL:  config.GetConfigDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: config.GetConfigDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),

		StorageProvider: GetConfigVal("STORAGE_PROVIDER", "local"),
		LocalStorageDir: GetConfigVal("LOCAL_STORAGE_DIR", ".data/uploads"),
		CloudinaryURL:   GetConfigVal("CLOUDINARY_URL", ""),
		MaxUploadMB:     config.GetConfigInt64("MAX_UPLOAD_MB", 20),

		Extractor:         GetConfigVal("EXTRACTOR", "gemini"),
		GeminiAPIKey:      GetConfigVal("GEMINI_API_KEY", ""),
		GeminiModel:       GetConfigVal("GEMINI_MODEL", "gemini-3.1-pro-preview"),
		ExtractionWorkers: config.GetConfigInt("EXTRACTION_WORKERS", 2),

		OAuth: auth.OAuthConfig{
			GoogleClientID:        GetConfigVal("OAUTH_GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret:    GetConfigVal("OAUTH_GOOGLE_CLIENT_SECRET", ""),
			MicrosoftClientID:     GetConfigVal("OAUTH_MICROSOFT_CLIENT_ID", ""),
			MicrosoftClientSecret: GetConfigVal("OAUTH_MICROSOFT_CLIENT_SECRET", ""),
			MicrosoftTenant:       GetConfigVal("OAUTH_MICROSOFT_TENANT", "common"),
			AppleClientID:         GetConfigVal("OAUTH_APPLE_CLIENT_ID", ""),
			AppleTeamID:           GetConfigVal("OAUTH_APPLE_TEAM_ID", ""),
			AppleKeyID:            GetConfigVal("OAUTH_APPLE_KEY_ID", ""),
			ApplePrivateKey:       GetConfigVal("OAUTH_APPLE_PRIVATE_KEY", ""),
		},
		Mail: mail.Config{
			Provider: GetConfigVal("MAIL_PROVIDER", "log"),
			From:     GetConfigVal("MAIL_FROM", "ExpenseLens <no-reply@expenselens.local>"),
			SMTPHost: GetConfigVal("SMTP_HOST", ""),
			SMTPPort: config.GetConfigInt("SMTP_PORT", 587),
			SMTPUser: GetConfigVal("SMTP_USER", ""),
			SMTPPass: GetConfigVal("SMTP_PASS", ""),
		},
		ConfigReloadSecret: GetConfigVal("CONFIG_RELOAD_SECRET", ""),
	}
}

// ExtractionConfig adapts AppConfig for the extraction package.
func (c *Config) ExtractionConfig() extraction.Config {
	return extraction.Config{Extractor: c.Extractor, APIKey: c.GeminiAPIKey, Model: c.GeminiModel}
}

// ValidateProductionSecrets fails fast on placeholder secrets in production.
// Call after Reload so DB-backed values are considered.
func ValidateProductionSecrets() error {
	if AppConfig == nil || !AppConfig.IsProd() {
		return nil
	}
	if AppConfig.JWTSecret == "" || AppConfig.JWTSecret == config.DevJWTSecret {
		return errors.New("config: JWT_SECRET must be set to a non-default value when ENV=production")
	}
	return nil
}
