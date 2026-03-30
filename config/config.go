package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port           string
	DataDir        string
	DBPath         string
	JWTSecret      string
	JWTExpiry      time.Duration
	RefreshExpiry  time.Duration
	UploadMaxBytes int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		DataDir:        getEnv("DATA_DIR", "./data"),
		DBPath:         getEnv("DB_PATH", "./data/alexandria.db"),
		JWTSecret:      getEnv("JWT_SECRET", "changeme-please-set-in-production"),
		JWTExpiry:      getDuration("JWT_EXPIRY", 15*time.Minute),
		RefreshExpiry:  getDuration("REFRESH_EXPIRY", 168*time.Hour),
		UploadMaxBytes: getInt("UPLOAD_MAX_BYTES", 64*1024*1024),
	}
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
