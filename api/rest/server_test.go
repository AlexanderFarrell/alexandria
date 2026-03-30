package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"alexandria/app"
	"alexandria/config"
	"alexandria/infra/epub"
	"alexandria/infra/sqlite"
	"alexandria/infra/storage"
)

func TestUploadAllowsLargeMultipartBooks(t *testing.T) {
	srv, cfg := newTestServer(t, 8*1024*1024)
	token := registerTestUser(t, srv, "reader_large_upload")

	largePayload := bytes.Repeat([]byte("x"), 6*1024*1024)
	req := newUploadRequest(t, largePayload, "large-upload.epub", token)

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("upload large book: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201, got %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Book struct {
			Title    string `json:"title"`
			FileType string `json:"file_type"`
			FilePath string `json:"file_path"`
		} `json:"book"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	if payload.Book.Title != "large-upload" {
		t.Fatalf("expected fallback title %q, got %q", "large-upload", payload.Book.Title)
	}
	if payload.Book.FileType != "epub" {
		t.Fatalf("expected file type epub, got %q", payload.Book.FileType)
	}

	fullPath := filepath.Join(cfg.DataDir, payload.Book.FilePath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("uploaded file was not saved: %v", err)
	}
}

func TestUploadRejectsMultipartBooksOverConfiguredLimit(t *testing.T) {
	srv, _ := newTestServer(t, 1*1024*1024)
	token := registerTestUser(t, srv, "reader_limit_rejection")
	baseURL := startTestHTTPServer(t, srv)

	oversizedPayload := bytes.Repeat([]byte("x"), 2*1024*1024)
	req := newLiveUploadRequest(t, baseURL, oversizedPayload, "too-large.epub", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload oversized book: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 413, got %d: %s", resp.StatusCode, body)
	}
}

func newTestServer(t *testing.T, uploadMaxBytes int) (*FiberServer, *config.Config) {
	t.Helper()

	baseDir := t.TempDir()
	cfg := &config.Config{
		Port:           "0",
		DataDir:        filepath.Join(baseDir, "data"),
		DBPath:         filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:      "test-secret",
		JWTExpiry:      15 * time.Minute,
		RefreshExpiry:  24 * time.Hour,
		UploadMaxBytes: uploadMaxBytes,
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	application := app.New(
		sqlite.NewUserRepo(db),
		sqlite.NewBookRepo(db),
		sqlite.NewProgressRepo(db),
		sqlite.NewListRepo(db),
		storage.NewLocalFileStore(cfg.DataDir),
		epub.New(),
		cfg,
	)

	return New(application, "", cfg.UploadMaxBytes), cfg
}

func startTestHTTPServer(t *testing.T, srv *FiberServer) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on test port: %v", err)
	}

	go func() {
		if err := srv.app.Listener(ln); err != nil {
			t.Logf("test fiber server stopped: %v", err)
		}
	}()

	t.Cleanup(func() {
		if err := srv.Shutdown(); err != nil {
			t.Logf("shutdown test server: %v", err)
		}
	})

	return "http://" + ln.Addr().String()
}

func registerTestUser(t *testing.T, srv *FiberServer, username string) string {
	t.Helper()

	body := bytes.NewBufferString(`{"username":"` + username + `","password":"testpass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("register test user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201, got %d: %s", resp.StatusCode, respBody)
	}

	var payload struct {
		Tokens struct {
			AccessToken string `json:"access_token"`
		} `json:"tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if payload.Tokens.AccessToken == "" {
		t.Fatal("register response did not include an access token")
	}
	return payload.Tokens.AccessToken
}

func newUploadRequest(t *testing.T, content []byte, filename, token string) *http.Request {
	t.Helper()

	body, contentType := newUploadBody(t, content, filename)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/books", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func newLiveUploadRequest(t *testing.T, baseURL string, content []byte, filename, token string) *http.Request {
	t.Helper()

	body, contentType := newUploadBody(t, content, filename)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/books", body)
	if err != nil {
		t.Fatalf("create live upload request: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func newUploadBody(t *testing.T, content []byte, filename string) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart file part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	return &body, writer.FormDataContentType()
}
