package rest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"alexandria/app"
	"alexandria/config"
	"alexandria/domain"
	"alexandria/infra/epub"
	"alexandria/infra/sqlite"
	"alexandria/infra/storage"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func TestUploadAllowsLargeMultipartBooks(t *testing.T) {
	srv, cfg := newTestServer(t, 8*1024*1024)
	token := registerTestUser(t, srv, "reader_large_upload")

	largePayload := fakeEPUBPayload(6 * 1024 * 1024)
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

	oversizedPayload := fakeEPUBPayload(2 * 1024 * 1024)
	req := newLiveUploadRequest(t, baseURL, oversizedPayload, "too-large.epub", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection reset by peer") {
			return
		}
		t.Fatalf("upload oversized book: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 413, got %d: %s", resp.StatusCode, body)
	}
}

func TestUploadRejectsInvalidEPUBSignature(t *testing.T) {
	srv, _ := newTestServer(t, 8*1024*1024)
	token := registerTestUser(t, srv, "reader_invalid_signature")

	req := newUploadRequest(t, []byte("not a zip file"), "bad.epub", token)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("upload invalid epub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 400, got %d: %s", resp.StatusCode, body)
	}
}

func TestAuthStatusSingleModeAllowsBootstrapOnly(t *testing.T) {
	srv, _ := newTestServerWithMode(t, config.RegistrationModeSingle, nil)

	status := getAuthStatus(t, srv)
	if status.RegistrationMode != string(config.RegistrationModeSingle) || !status.CanRegister {
		t.Fatalf("unexpected initial auth status: %+v", status)
	}

	registerTestUser(t, srv, "owner_account")

	status = getAuthStatus(t, srv)
	if status.CanRegister {
		t.Fatalf("expected registration to be closed after bootstrap: %+v", status)
	}

	body := bytes.NewBufferString(`{"username":"another","password":"testpass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("second registration request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 403, got %d: %s", resp.StatusCode, payload)
	}
}

func TestAuthStatusDisableModeRejectsRegistration(t *testing.T) {
	srv, _ := newTestServerWithMode(t, config.RegistrationModeDisable, nil)

	status := getAuthStatus(t, srv)
	if status.RegistrationMode != string(config.RegistrationModeDisable) || status.CanRegister {
		t.Fatalf("unexpected disabled auth status: %+v", status)
	}

	body := bytes.NewBufferString(`{"username":"owner","password":"testpass123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("disabled registration request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 403, got %d: %s", resp.StatusCode, payload)
	}
}

func TestReadyzReportsReadyStatus(t *testing.T) {
	srv, _ := newTestServer(t, 8*1024*1024)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("readyz request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestReadyzReturnsServiceUnavailableWhenCheckFails(t *testing.T) {
	srv, _ := newTestServerWithMode(t, config.RegistrationModeSingle, func() error {
		return errors.New("db unavailable at /tmp/secret.db")
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("failing readyz request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 503, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	if bytes.Contains(body, []byte("/tmp/secret.db")) {
		t.Fatalf("readiness response leaked internal details: %s", body)
	}
}

func TestInternalErrorsAreRedacted(t *testing.T) {
	srv, _ := newTestServer(t, 8*1024*1024)
	srv.app.Get("/boom", func(c *fiber.Ctx) error {
		return errors.New("db failed at /tmp/secret.db")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("boom request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 500, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	if bytes.Contains(body, []byte("/tmp/secret.db")) {
		t.Fatalf("500 response leaked internals: %s", body)
	}
	if !bytes.Contains(body, []byte("internal server error")) {
		t.Fatalf("expected generic 500 response, got %s", body)
	}
}

func TestMCPRouteDisabledWithoutToken(t *testing.T) {
	srv, _ := newTestServer(t, 8*1024*1024)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("disabled mcp request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 404, got %d: %s", resp.StatusCode, body)
	}
}

func TestMCPRouteRequiresBearerToken(t *testing.T) {
	baseDir := t.TempDir()
	srv, _, _ := newTestServerWithConfig(t, &config.Config{
		Port:             "0",
		DataDir:          filepath.Join(baseDir, "data"),
		DBPath:           filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   8 * 1024 * 1024,
		RegistrationMode: config.RegistrationModeSingle,
		MCPHTTPToken:     "mcp-secret",
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("unauthenticated mcp request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 401, got %d: %s", resp.StatusCode, body)
	}

	authReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{}`))
	authReq.Header.Set("Content-Type", "application/json")
	authReq.Header.Set("Authorization", "Bearer mcp-secret")

	authResp, err := srv.app.Test(authReq, -1)
	if err != nil {
		t.Fatalf("authenticated mcp request: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode == http.StatusUnauthorized || authResp.StatusCode == http.StatusNotFound {
		body, _ := io.ReadAll(authResp.Body)
		t.Fatalf("expected authenticated MCP request to reach the handler, got %d: %s", authResp.StatusCode, body)
	}
}

func TestServeCoverStreamsDetectedContentType(t *testing.T) {
	baseDir := t.TempDir()
	srv, cfg, db := newTestServerWithConfig(t, &config.Config{
		Port:             "0",
		DataDir:          filepath.Join(baseDir, "data"),
		DBPath:           filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   8 * 1024 * 1024,
		RegistrationMode: config.RegistrationModeSingle,
	}, nil)
	token := registerTestUser(t, srv, "reader_cover_stream")

	coverPath := filepath.Join("books", "cover-book", "cover.jpg")
	fullCoverPath := filepath.Join(cfg.DataDir, coverPath)
	if err := os.MkdirAll(filepath.Dir(fullCoverPath), 0o755); err != nil {
		t.Fatalf("mkdir cover dir: %v", err)
	}
	coverBytes := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(fullCoverPath, coverBytes, 0o644); err != nil {
		t.Fatalf("write cover file: %v", err)
	}

	book := &domain.Book{
		ID:        "cover-book",
		Title:     "Cover Book",
		Author:    "Test Author",
		FilePath:  filepath.Join("books", "cover-book", "original.epub"),
		CoverPath: coverPath,
		FileType:  domain.FileTypeEPUB,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sqlite.NewBookRepo(db).Create(context.Background(), book); err != nil {
		t.Fatalf("create cover book: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books/cover-book/cover", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("cover request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png content type, got %q", got)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Equal(body, coverBytes) {
		t.Fatal("cover response body did not match stored cover bytes")
	}
}

func TestReaderManifestAndSectionRequireAuth(t *testing.T) {
	srv, _ := newTestServer(t, 8*1024*1024)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/books/some-book/reader/manifest", nil)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("manifest request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 401 for unauthenticated manifest, got %d: %s", resp.StatusCode, body)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/books/some-book/reader/sections/chap-1", nil)
	resp, err = srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("section request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 401 for unauthenticated section, got %d: %s", resp.StatusCode, body)
	}
}

func TestReaderEndpointsServeManifestSectionsAndSignedAssets(t *testing.T) {
	baseDir := t.TempDir()
	srv, cfg, db := newTestServerWithConfig(t, &config.Config{
		Port:             "0",
		DataDir:          filepath.Join(baseDir, "data"),
		DBPath:           filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   8 * 1024 * 1024,
		RegistrationMode: config.RegistrationModeSingle,
	}, nil)
	token := registerTestUser(t, srv, "reader_manifest_user")

	createStoredEPUBBook(t, cfg, db, "reader-book", buildReaderFixtureEPUBForServer(t))
	createStoredEPUBBook(t, cfg, db, "reader-book-two", buildReaderFixtureEPUBForServer(t))

	manifestReq := httptest.NewRequest(http.MethodGet, "/api/v1/books/reader-book/reader/manifest", nil)
	manifestReq.Header.Set("Authorization", "Bearer "+token)
	manifestResp, err := srv.app.Test(manifestReq, -1)
	if err != nil {
		t.Fatalf("manifest request: %v", err)
	}
	defer manifestResp.Body.Close()

	if manifestResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(manifestResp.Body)
		t.Fatalf("expected status 200, got %d: %s", manifestResp.StatusCode, body)
	}

	var manifestPayload struct {
		Manifest struct {
			FirstSectionID string `json:"first_section_id"`
			Sections       []struct {
				ID string `json:"id"`
			} `json:"sections"`
		} `json:"manifest"`
	}
	if err := json.NewDecoder(manifestResp.Body).Decode(&manifestPayload); err != nil {
		t.Fatalf("decode manifest response: %v", err)
	}
	if manifestPayload.Manifest.FirstSectionID != "chap-1" {
		t.Fatalf("expected first section chap-1, got %q", manifestPayload.Manifest.FirstSectionID)
	}
	if len(manifestPayload.Manifest.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(manifestPayload.Manifest.Sections))
	}

	sectionReq := httptest.NewRequest(http.MethodGet, "/api/v1/books/reader-book/reader/sections/chap-1", nil)
	sectionReq.Header.Set("Authorization", "Bearer "+token)
	sectionResp, err := srv.app.Test(sectionReq, -1)
	if err != nil {
		t.Fatalf("section request: %v", err)
	}
	defer sectionResp.Body.Close()

	if sectionResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sectionResp.Body)
		t.Fatalf("expected status 200, got %d: %s", sectionResp.StatusCode, body)
	}

	var sectionPayload struct {
		Section struct {
			HTML string `json:"html"`
		} `json:"section"`
	}
	if err := json.NewDecoder(sectionResp.Body).Decode(&sectionPayload); err != nil {
		t.Fatalf("decode section response: %v", err)
	}

	if !strings.Contains(sectionPayload.Section.HTML, `data-reader-section-id="chap-2"`) {
		t.Fatalf("expected cross-section link rewrite, got %s", sectionPayload.Section.HTML)
	}

	stylesheetURL := extractReaderAssetURL(t, sectionPayload.Section.HTML, `(/api/v1/books/reader-book/reader/assets/OEBPS/styles/book\.css\?rt=[^"]+)`)
	assetReq := httptest.NewRequest(http.MethodGet, stylesheetURL, nil)
	assetResp, err := srv.app.Test(assetReq, -1)
	if err != nil {
		t.Fatalf("stylesheet asset request: %v", err)
	}
	defer assetResp.Body.Close()

	if assetResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(assetResp.Body)
		t.Fatalf("expected status 200 for signed stylesheet asset, got %d: %s", assetResp.StatusCode, body)
	}
	if got := assetResp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("expected css content type, got %q", got)
	}

	cssBody, _ := io.ReadAll(assetResp.Body)
	imageURL := extractReaderAssetURL(t, string(cssBody), `(/api/v1/books/reader-book/reader/assets/OEBPS/images/pic\.png\?rt=[^")]+)`)
	imageReq := httptest.NewRequest(http.MethodGet, imageURL, nil)
	imageResp, err := srv.app.Test(imageReq, -1)
	if err != nil {
		t.Fatalf("image asset request: %v", err)
	}
	defer imageResp.Body.Close()

	if imageResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(imageResp.Body)
		t.Fatalf("expected status 200 for signed image asset, got %d: %s", imageResp.StatusCode, body)
	}
	if got := imageResp.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png content type, got %q", got)
	}

	wrongBookReq := httptest.NewRequest(http.MethodGet, strings.Replace(stylesheetURL, "/reader-book/", "/reader-book-two/", 1), nil)
	wrongBookResp, err := srv.app.Test(wrongBookReq, -1)
	if err != nil {
		t.Fatalf("wrong-book asset request: %v", err)
	}
	defer wrongBookResp.Body.Close()

	if wrongBookResp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(wrongBookResp.Body)
		t.Fatalf("expected status 401 for wrong-book asset token, got %d: %s", wrongBookResp.StatusCode, body)
	}

	traversalURL := strings.Replace(stylesheetURL, "OEBPS/styles/book.css", "%2E%2E/%2E%2E/META-INF/container.xml", 1)
	traversalReq := httptest.NewRequest(http.MethodGet, traversalURL, nil)
	traversalResp, err := srv.app.Test(traversalReq, -1)
	if err != nil {
		t.Fatalf("traversal asset request: %v", err)
	}
	defer traversalResp.Body.Close()

	if traversalResp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(traversalResp.Body)
		t.Fatalf("expected status 404 for traversal asset request, got %d: %s", traversalResp.StatusCode, body)
	}
}

func TestSPAFallbackServesIndexOnlyForHTMLNavigations(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!DOCTYPE html><title>Alexandria</title>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	srv, _ := newStaticTestServer(t, staticDir)

	req := httptest.NewRequest(http.MethodGet, "/library", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("spa navigation request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected no-cache for index.html, got %q", got)
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Alexandria")) {
		t.Fatalf("expected SPA shell body, got %s", body)
	}
}

func TestSPAFallbackDoesNotServeIndexForMissingAssets(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!DOCTYPE html><title>Alexandria</title>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	srv, _ := newStaticTestServer(t, staticDir)

	req := httptest.NewRequest(http.MethodGet, "/assets/index-oldhash.js", nil)
	req.Header.Set("Accept", "*/*")

	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("missing asset request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 404, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	if bytes.Contains(body, []byte("Alexandria")) {
		t.Fatalf("expected missing asset to avoid SPA fallback, got %s", body)
	}
}

func TestStaticAssetCacheHeaders(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(staticDir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!DOCTYPE html><title>Alexandria</title>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "sw.js"), []byte("self.addEventListener('install', () => {})"), 0o644); err != nil {
		t.Fatalf("write sw.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "assets", "app.js"), []byte("console.log('app')"), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}

	srv, _ := newStaticTestServer(t, staticDir)

	swReq := httptest.NewRequest(http.MethodGet, "/sw.js", nil)
	swResp, err := srv.app.Test(swReq, -1)
	if err != nil {
		t.Fatalf("sw request: %v", err)
	}
	defer swResp.Body.Close()

	if got := swResp.Header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("expected no-cache for sw.js, got %q", got)
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	assetResp, err := srv.app.Test(assetReq, -1)
	if err != nil {
		t.Fatalf("asset request: %v", err)
	}
	defer assetResp.Body.Close()

	if got := assetResp.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("expected immutable cache header for built assets, got %q", got)
	}
}

func newTestServer(t *testing.T, uploadMaxBytes int) (*FiberServer, *config.Config) {
	t.Helper()

	baseDir := t.TempDir()
	srv, cfg, _ := newTestServerWithConfig(t, &config.Config{
		Port:             "0",
		DataDir:          filepath.Join(baseDir, "data"),
		DBPath:           filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   uploadMaxBytes,
		RegistrationMode: config.RegistrationModeSingle,
	}, nil)
	return srv, cfg
}

func newTestServerWithMode(t *testing.T, mode config.RegistrationMode, ready func() error) (*FiberServer, *config.Config) {
	t.Helper()

	baseDir := t.TempDir()
	srv, cfg, _ := newTestServerWithConfig(t, &config.Config{
		Port:             "0",
		DataDir:          filepath.Join(baseDir, "data"),
		DBPath:           filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   8 * 1024 * 1024,
		RegistrationMode: mode,
	}, ready)
	return srv, cfg
}

func newTestServerWithConfig(t *testing.T, cfg *config.Config, ready func() error) (*FiberServer, *config.Config, *gorm.DB) {
	t.Helper()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
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
		nil,
		cfg,
	)

	return New(application, "", cfg, ready), cfg, db
}

func newStaticTestServer(t *testing.T, staticDir string) (*FiberServer, *config.Config) {
	t.Helper()

	baseDir := t.TempDir()
	cfg := &config.Config{
		Port:             "0",
		DataDir:          filepath.Join(baseDir, "data"),
		DBPath:           filepath.Join(baseDir, "data", "alexandria.db"),
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   8 * 1024 * 1024,
		RegistrationMode: config.RegistrationModeSingle,
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		t.Fatalf("create data dir: %v", err)
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
		nil,
		cfg,
	)

	return New(application, staticDir, cfg, nil), cfg
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

func getAuthStatus(t *testing.T, srv *FiberServer) struct {
	RegistrationMode string `json:"registration_mode"`
	CanRegister      bool   `json:"can_register"`
} {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	resp, err := srv.app.Test(req, -1)
	if err != nil {
		t.Fatalf("auth status request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		RegistrationMode string `json:"registration_mode"`
		CanRegister      bool   `json:"can_register"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode auth status response: %v", err)
	}
	return payload
}

func buildReaderFixtureEPUBForServer(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	writeZipEntry(t, zipWriter, "mimetype", []byte("application/epub+zip"))
	writeZipEntry(t, zipWriter, "META-INF/container.xml", []byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))
	writeZipEntry(t, zipWriter, "OEBPS/content.opf", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="BookId">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Reader Fixture</dc:title>
    <dc:creator>Fixture Author</dc:creator>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="css" href="styles/book.css" media-type="text/css"/>
    <item id="img" href="images/pic.png" media-type="image/png"/>
    <item id="chap-1" href="text/ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chap-2" href="text/ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="chap-3" href="text/ch3.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="chap-1"/>
    <itemref idref="chap-2"/>
    <itemref idref="chap-3"/>
  </spine>
</package>`))
	writeZipEntry(t, zipWriter, "OEBPS/nav.xhtml", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <head><title>Nav</title></head>
  <body>
    <nav epub:type="toc">
      <ol>
        <li><a href="text/ch2.xhtml">Second Chapter</a></li>
        <li><a href="text/ch1.xhtml">First Chapter</a></li>
      </ol>
    </nav>
  </body>
</html>`))
	writeZipEntry(t, zipWriter, "OEBPS/toc.ncx", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="one" playOrder="1">
      <navLabel><text>First Chapter</text></navLabel>
      <content src="text/ch1.xhtml"/>
    </navPoint>
    <navPoint id="two" playOrder="2">
      <navLabel><text>Second Chapter</text></navLabel>
      <content src="text/ch2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`))
	writeZipEntry(t, zipWriter, "OEBPS/styles/book.css", []byte(`body { background-image: url('../images/pic.png'); }`))
	writeZipEntry(t, zipWriter, "OEBPS/images/pic.png", []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	})
	writeZipEntry(t, zipWriter, "OEBPS/text/ch1.xhtml", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Chapter One</title>
    <link rel="stylesheet" href="../styles/book.css"/>
  </head>
  <body>
    <h1>Chapter One</h1>
    <p><a href="ch2.xhtml#part-two">Next section</a></p>
  </body>
</html>`))
	writeZipEntry(t, zipWriter, "OEBPS/text/ch2.xhtml", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter Two</title></head>
  <body>
    <h1 id="part-two">Chapter Two</h1>
    <p>Two.</p>
  </body>
</html>`))
	writeZipEntry(t, zipWriter, "OEBPS/text/ch3.xhtml", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter Three</title></head>
  <body>
    <h1>Chapter Three</h1>
  </body>
</html>`))

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buffer.Bytes()
}

func writeZipEntry(t *testing.T, writer *zip.Writer, name string, payload []byte) {
	t.Helper()
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %s: %v", name, err)
	}
	if _, err := entry.Write(payload); err != nil {
		t.Fatalf("write zip entry %s: %v", name, err)
	}
}

func createStoredEPUBBook(t *testing.T, cfg *config.Config, db *gorm.DB, bookID string, payload []byte) {
	t.Helper()
	filePath := filepath.Join("books", bookID, "original.epub")
	fullPath := filepath.Join(cfg.DataDir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir book dir: %v", err)
	}
	if err := os.WriteFile(fullPath, payload, 0o644); err != nil {
		t.Fatalf("write epub fixture: %v", err)
	}

	book := &domain.Book{
		ID:        bookID,
		Title:     "Fixture Book",
		Author:    "Fixture Author",
		FilePath:  filePath,
		FileType:  domain.FileTypeEPUB,
		FileSize:  int64(len(payload)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sqlite.NewBookRepo(db).Create(context.Background(), book); err != nil {
		t.Fatalf("create book fixture: %v", err)
	}
}

func extractReaderAssetURL(t *testing.T, body string, pattern string) string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(body)
	if len(match) < 2 {
		t.Fatalf("failed to find asset url with pattern %q in %s", pattern, body)
	}
	return match[1]
}

func fakeEPUBPayload(size int) []byte {
	if size < 4 {
		size = 4
	}
	payload := bytes.Repeat([]byte("x"), size)
	copy(payload, []byte("PK\x03\x04"))
	return payload
}
