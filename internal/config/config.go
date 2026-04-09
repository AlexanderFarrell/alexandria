package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultJWTSecretPlaceholder = "changeme-please-set-in-production"
	minJWTSecretLength          = 32
)

// RegistrationMode controls whether self-service account creation is allowed.
type RegistrationMode string

const (
	RegistrationModeDisable RegistrationMode = "disable"
	RegistrationModeSingle  RegistrationMode = "single"
	RegistrationModeMulti   RegistrationMode = "multi"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port             string
	DataDir          string
	DBPath           string
	JWTSecret        string
	JWTExpiry        time.Duration
	RefreshExpiry    time.Duration
	UploadMaxBytes   int
	RegistrationMode RegistrationMode
	CORSAllowOrigins []string
	MCPHTTPToken     string
	MCPOwnerUsername string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		DataDir:          getEnv("DATA_DIR", "./data"),
		DBPath:           getEnv("DB_PATH", "./data/alexandria.db"),
		JWTSecret:        getEnv("JWT_SECRET", defaultJWTSecretPlaceholder),
		JWTExpiry:        getDuration("JWT_EXPIRY", 15*time.Minute),
		RefreshExpiry:    getDuration("REFRESH_EXPIRY", 168*time.Hour),
		UploadMaxBytes:   getInt("UPLOAD_MAX_BYTES", 500*1024*1024),
		RegistrationMode: getRegistrationMode("REGISTRATION_MODE", RegistrationModeSingle),
		CORSAllowOrigins: getCSV("CORS_ALLOW_ORIGINS"),
		MCPHTTPToken:     strings.TrimSpace(os.Getenv("MCP_HTTP_TOKEN")),
		MCPOwnerUsername: strings.TrimSpace(os.Getenv("MCP_OWNER_USERNAME")),
	}
}

// Validate checks that required runtime configuration is safe for startup.
func (c *Config) Validate() error {
	secret := strings.TrimSpace(c.JWTSecret)
	switch {
	case secret == "":
		return fmt.Errorf("JWT_SECRET must be set")
	case secret == defaultJWTSecretPlaceholder:
		return fmt.Errorf("JWT_SECRET must not use the default placeholder value")
	case len(secret) < minJWTSecretLength:
		return fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}

	switch c.RegistrationMode {
	case RegistrationModeDisable, RegistrationModeSingle, RegistrationModeMulti:
	default:
		return fmt.Errorf("REGISTRATION_MODE must be one of: disable, single, multi")
	}

	for _, origin := range c.CORSAllowOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOW_ORIGINS must be an explicit allowlist; wildcard is not supported")
		}
	}

	if strings.TrimSpace(c.MCPOwnerUsername) != c.MCPOwnerUsername {
		return fmt.Errorf("MCP_OWNER_USERNAME must not contain leading or trailing whitespace")
	}

	if strings.TrimSpace(c.MCPHTTPToken) != c.MCPHTTPToken {
		return fmt.Errorf("MCP_HTTP_TOKEN must not contain leading or trailing whitespace")
	}

	return nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}

func getInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

func getRegistrationMode(key string, defaultVal RegistrationMode) RegistrationMode {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return defaultVal
	}
	return RegistrationMode(v)
}

func getCSV(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}

	parts := strings.Split(v, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			values = append(values, item)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
