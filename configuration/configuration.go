package configuration

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment or .env file.
// Prefer JWT_SECRET and SERVER_ADDRESS; Key/Secret are supported for backward compatibility.
type Config struct {
	// Database: either DBSource (single URL) or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/DB_TYPE
	DBSource string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBType     string

	// Auth / JWT (env: JWT_SECRET or Secret)
	JWTSecret string

	// Server (env: SERVER_ADDRESS; default ":8080")
	ServerAddress string

	// Migrations (env: MIGRATION_URL; default "file://db/migrations"). Used when running migrations at startup or via CLI.
	MigrationURL string

	// Redis (env: REDIS_URL — e.g. "redis://localhost:6379"). Optional: if unset, in-memory cache is used.
	RedisURL string

	// Optional (env: ENVIRONMENT — e.g. "release", "debug", "development", "test")
	Environment string
	Version     string
}

// LoadConfig loads configuration from environment variables, optionally from a .env file if present.
// If no .env file exists (e.g. in containers with env-only config), loading is skipped and env vars are used.
// Returns (Config, error). JWT_SECRET is preferred; falls back to "Secret" for backward compatibility.
// SERVER_ADDRESS defaults to ":8080" if unset.
func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		// Optional file: allow env-only (e.g. Docker, 12-factor); only fail on real load errors
		if !isNotFound(err) {
			return Config{}, fmt.Errorf("loading .env: %w", err)
		}
	}

	cfg := Config{
		DBSource:      os.Getenv("DB_SOURCE"),
		DBHost:        os.Getenv("DB_HOST"),
		DBPort:        os.Getenv("DB_PORT"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		DBType:        os.Getenv("DB_TYPE"),
		ServerAddress: os.Getenv("SERVER_ADDRESS"),
		MigrationURL:  os.Getenv("MIGRATION_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		Environment:   os.Getenv("ENVIRONMENT"),
		Version:       os.Getenv("Version"),
	}
	if cfg.DBSource == "" {
		cfg.DBSource = os.Getenv("DATABASE_URL")
	}

	if cfg.ServerAddress == "" {
		cfg.ServerAddress = ":8080"
	}
	if cfg.MigrationURL == "" {
		cfg.MigrationURL = "file://db/migrations"
	}

	// JWT secret: prefer JWT_SECRET, fallback to legacy "Secret"
	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = os.Getenv("Secret")
	}

	if err := validateRequired(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

const minJWTSecretLength = 32

func validateRequired(cfg Config) error {
	if cfg.JWTSecret == "" {
		return fmt.Errorf("missing required config: JWT_SECRET (or Secret)")
	}
	if len(cfg.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}
	// Database: either DBSource or all DB_* vars
	if cfg.DBSource != "" {
		return nil
	}
	required := []struct {
		name  string
		value string
	}{
		{"DB_HOST", cfg.DBHost},
		{"DB_PORT", cfg.DBPort},
		{"DB_USER", cfg.DBUser},
		{"DB_PASSWORD", cfg.DBPassword},
		{"DB_NAME", cfg.DBName},
		{"DB_TYPE", cfg.DBType},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config (set DB_SOURCE/DATABASE_URL or all of): %s", strings.Join(missing, ", "))
	}
	return nil
}

// isNotFound reports whether the error is due to .env file not existing (so env-only config is allowed).
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	// PathError from os.Open often wraps the message
	return strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "cannot find")
}

// DatabaseURL returns a single connection URL for the configured database, for use by the migration
// runner or other tools. When DBSource (or DATABASE_URL) is set, it is returned as-is. Otherwise
// the URL is built from DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, and DB_TYPE (postgres or sqlserver).
func DatabaseURL(c Config) string {
	if c.DBSource != "" {
		return c.DBSource
	}
	switch strings.ToLower(c.DBType) {
	case "postgres", "postgresql":
		u := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(c.DBUser, c.DBPassword),
			Host:     c.DBHost + ":" + c.DBPort,
			Path:     "/" + c.DBName,
			RawQuery: "sslmode=disable",
		}
		return u.String()
	case "sqlserver":
		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(c.DBUser, c.DBPassword),
			Host:     c.DBHost + ":" + c.DBPort,
			RawQuery: "database=" + url.QueryEscape(c.DBName),
		}
		return u.String()
	default:
		return ""
	}
}
