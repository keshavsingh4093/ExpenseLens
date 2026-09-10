// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds every setting the API needs. Values come from the environment;
// see .env.example for the full list.
type Config struct {
	Env     string `env:"APP_ENV" envDefault:"development"`
	Port    int    `env:"PORT" envDefault:"8090"`
	AppURL  string `env:"APP_URL" envDefault:"http://localhost:5173"`
	APIURL  string `env:"API_URL" envDefault:"http://localhost:8090"`
	LogJSON bool   `env:"LOG_JSON" envDefault:"false"`

	MongoURI string `env:"MONGO_URI" envDefault:"mongodb://localhost:27017"`
	MongoDB  string `env:"MONGO_DB" envDefault:"expenselens"`

	JWTSecret       string        `env:"JWT_SECRET" envDefault:"dev-secret-change-me"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`

	Storage StorageConfig
	Gemini  GeminiConfig
	OAuth   OAuthConfig
	Mail    MailConfig
}

// StorageConfig selects and configures the file storage provider.
type StorageConfig struct {
	Provider      string `env:"STORAGE_PROVIDER" envDefault:"local"` // local | cloudinary
	LocalDir      string `env:"LOCAL_STORAGE_DIR" envDefault:".data/uploads"`
	CloudinaryURL string `env:"CLOUDINARY_URL"` // cloudinary://key:secret@cloud
	MaxUploadMB   int64  `env:"MAX_UPLOAD_MB" envDefault:"20"`
}

// GeminiConfig configures the receipt extraction model.
type GeminiConfig struct {
	APIKey string `env:"GEMINI_API_KEY"`
	Model  string `env:"GEMINI_MODEL" envDefault:"gemini-3.1-pro-preview"`
	// Extractor selects the implementation: gemini | fake (dev/test).
	Extractor string `env:"EXTRACTOR" envDefault:"gemini"`
	Workers   int    `env:"EXTRACTION_WORKERS" envDefault:"2"`
}

// OAuthConfig holds credentials for each social login provider. A provider is
// enabled only when its client id is non-empty.
type OAuthConfig struct {
	GoogleClientID     string `env:"OAUTH_GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"OAUTH_GOOGLE_CLIENT_SECRET"`

	MicrosoftClientID     string `env:"OAUTH_MICROSOFT_CLIENT_ID"`
	MicrosoftClientSecret string `env:"OAUTH_MICROSOFT_CLIENT_SECRET"`
	MicrosoftTenant       string `env:"OAUTH_MICROSOFT_TENANT" envDefault:"common"`

	AppleClientID   string `env:"OAUTH_APPLE_CLIENT_ID"` // Services ID
	AppleTeamID     string `env:"OAUTH_APPLE_TEAM_ID"`
	AppleKeyID      string `env:"OAUTH_APPLE_KEY_ID"`
	ApplePrivateKey string `env:"OAUTH_APPLE_PRIVATE_KEY"` // PEM contents, \n-escaped allowed
}

// MailConfig configures outbound email. Provider "log" prints to stdout.
type MailConfig struct {
	Provider string `env:"MAIL_PROVIDER" envDefault:"log"` // log | smtp
	From     string `env:"MAIL_FROM" envDefault:"ExpenseLens <no-reply@expenselens.local>"`
	SMTPHost string `env:"SMTP_HOST"`
	SMTPPort int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser string `env:"SMTP_USER"`
	SMTPPass string `env:"SMTP_PASS"`
}

// Load reads .env (if present) then the process environment.
func Load() (*Config, error) {
	_ = godotenv.Load() // best effort; real env always wins
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if c.IsProd() && c.JWTSecret == "dev-secret-change-me" {
		return nil, fmt.Errorf("config: JWT_SECRET must be set in production")
	}
	return &c, nil
}

// IsProd reports whether the app runs with APP_ENV=production.
func (c *Config) IsProd() bool { return c.Env == "production" }
