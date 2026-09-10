// Package config loads the bootstrap configuration from the environment and
// layers the dynamic system_config collection on top of it.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// BaseConfig holds the settings every backend needs to boot. Everything
// else comes from GetConfigVal (env → system_config → code default).
type BaseConfig struct {
	Env           string // development | production | test
	Port          int
	MongoURI      string
	MongoDatabase string
	JWTSecret     string
	LogJSON       bool
}

// DevJWTSecret is the placeholder accepted only outside production.
const DevJWTSecret = "dev-secret-change-me"

// LoadBase reads .env (bootstrap) then overlays ENV_FILE, or .env.local when
// present, and returns the base config. Real environment variables win.
func LoadBase(defaultPort int) *BaseConfig {
	_ = godotenv.Load()
	if f := os.Getenv("ENV_FILE"); f != "" {
		_ = godotenv.Overload(f)
	} else if _, err := os.Stat(".env.local"); err == nil {
		_ = godotenv.Overload(".env.local")
	}
	env := GetEnv("ENV", GetEnv("APP_ENV", "development"))
	secret := GetEnv("JWT_SECRET", "")
	if secret == "" {
		secret = DevJWTSecret
	}
	return &BaseConfig{
		Env:           env,
		Port:          GetEnvInt("PORT", defaultPort),
		MongoURI:      GetEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: GetEnv("MONGO_DATABASE", GetEnv("MONGO_DB", "expenselens")),
		JWTSecret:     secret,
		LogJSON:       GetEnvBool("LOG_JSON", false),
	}
}

// IsProd reports whether Env is production.
func (b *BaseConfig) IsProd() bool { return b.Env == "production" }

// GetEnv returns the environment value or fallback when unset/blank.
func GetEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// GetEnvInt parses an int from the environment.
func GetEnvInt(key string, fallback int) int {
	if v := GetEnv(key, ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// GetEnvBool parses a bool (true/1/yes/on) from the environment.
func GetEnvBool(key string, fallback bool) bool {
	return parseBool(GetEnv(key, ""), fallback)
}

func parseBool(v string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return fallback
}

// GetConfigVal resolves key from the environment first, then the dynamic
// system_config cache, then fallback.
func GetConfigVal(key, fallback string) string {
	if v := GetEnv(key, ""); v != "" {
		return v
	}
	return GetDynamic(key, fallback)
}

// GetConfigInt resolves an int through GetConfigVal.
func GetConfigInt(key string, fallback int) int {
	if v := GetConfigVal(key, ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// GetConfigInt64 resolves an int64 through GetConfigVal.
func GetConfigInt64(key string, fallback int64) int64 {
	if v := GetConfigVal(key, ""); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

// GetConfigBool resolves a bool through GetConfigVal.
func GetConfigBool(key string, fallback bool) bool {
	return parseBool(GetConfigVal(key, ""), fallback)
}

// GetConfigDuration resolves a Go duration ("15m") through GetConfigVal.
func GetConfigDuration(key string, fallback time.Duration) time.Duration {
	if v := GetConfigVal(key, ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
