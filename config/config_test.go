package config

import "testing"

func TestLoadDefaultsUploadMaxBytesTo500MiB(t *testing.T) {
	t.Setenv("UPLOAD_MAX_BYTES", "")

	cfg := Load()

	if cfg.UploadMaxBytes != 500*1024*1024 {
		t.Fatalf("expected default upload max bytes to be %d, got %d", 500*1024*1024, cfg.UploadMaxBytes)
	}
}

func TestLoadDefaultsRegistrationModeToSingle(t *testing.T) {
	t.Setenv("REGISTRATION_MODE", "")

	cfg := Load()

	if cfg.RegistrationMode != RegistrationModeSingle {
		t.Fatalf("expected default registration mode %q, got %q", RegistrationModeSingle, cfg.RegistrationMode)
	}
}

func TestLoadTrimsMCPConfig(t *testing.T) {
	t.Setenv("MCP_HTTP_TOKEN", "  token-value  ")
	t.Setenv("MCP_OWNER_USERNAME", "  owner  ")

	cfg := Load()

	if cfg.MCPHTTPToken != "token-value" {
		t.Fatalf("expected trimmed MCP token, got %q", cfg.MCPHTTPToken)
	}
	if cfg.MCPOwnerUsername != "owner" {
		t.Fatalf("expected trimmed MCP owner username, got %q", cfg.MCPOwnerUsername)
	}
}

func TestValidateRejectsDefaultJWTSecret(t *testing.T) {
	cfg := &Config{
		JWTSecret:        defaultJWTSecretPlaceholder,
		RegistrationMode: RegistrationModeSingle,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation to reject default JWT secret")
	}
}

func TestValidateRejectsWildcardCORS(t *testing.T) {
	cfg := &Config{
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		RegistrationMode: RegistrationModeSingle,
		CORSAllowOrigins: []string{"*"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation to reject wildcard CORS")
	}
}
