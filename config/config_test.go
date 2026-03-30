package config

import "testing"

func TestLoadDefaultsUploadMaxBytesTo500MiB(t *testing.T) {
	t.Setenv("UPLOAD_MAX_BYTES", "")

	cfg := Load()

	if cfg.UploadMaxBytes != 500*1024*1024 {
		t.Fatalf("expected default upload max bytes to be %d, got %d", 500*1024*1024, cfg.UploadMaxBytes)
	}
}
