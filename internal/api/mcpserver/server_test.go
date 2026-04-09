package mcpserver_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"alexandria/internal/api/mcpserver"
	"alexandria/internal/app"
	"alexandria/internal/app/ports"
	"alexandria/internal/config"
	"alexandria/internal/domain"
	"alexandria/internal/infra/epub"
	"alexandria/internal/infra/parser"
	"alexandria/internal/infra/pdf"
	"alexandria/internal/infra/sqlite"
	"alexandria/internal/infra/storage"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

type testRuntime struct {
	cfg         *config.Config
	db          *gorm.DB
	application *app.App
	server      *mcpserver.Server
}

func TestMCPServerToolsHappyPath(t *testing.T) {
	rt := newTestRuntime(t, config.RegistrationModeSingle, "", "")
	registerUser(t, rt, "owner")
	createStoredEPUBBook(t, rt, "reader-book", buildReaderFixtureEPUB(t))

	session := connectInMemory(t, rt.server)
	ctx := context.Background()

	searchRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "books_search",
	})
	if err != nil {
		t.Fatalf("books_search: %v", err)
	}
	search := decodeToolJSON[struct {
		Books []domain.Book `json:"books"`
		Total int64         `json:"total"`
		Page  int           `json:"page"`
		Limit int           `json:"limit"`
	}](t, searchRes)
	if search.Total != 1 || len(search.Books) != 1 || search.Page != 1 || search.Limit != 100 {
		t.Fatalf("unexpected books_search payload: %+v", search)
	}

	getRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "books_get",
		Arguments: map[string]any{"book_id": "reader-book"},
	})
	if err != nil {
		t.Fatalf("books_get: %v", err)
	}
	getBook := decodeToolJSON[struct {
		Book domain.Book `json:"book"`
	}](t, getRes)
	if getBook.Book.ID != "reader-book" {
		t.Fatalf("unexpected books_get payload: %+v", getBook)
	}

	updateRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "books_update",
		Arguments: map[string]any{
			"book_id":          "reader-book",
			"title":            "Updated Title",
			"author":           "Updated Author",
			"description":      "Updated Description",
			"zealot_ticket_id": "AX-50",
			"metadata": map[string]any{
				"genres":       []string{"Fiction"},
				"tags":         []string{"Priority"},
				"published_at": "2026-01-15",
			},
		},
	})
	if err != nil {
		t.Fatalf("books_update: %v", err)
	}
	updatedBook := decodeToolJSON[struct {
		Book domain.Book `json:"book"`
	}](t, updateRes)
	if updatedBook.Book.Title != "Updated Title" || updatedBook.Book.Author != "Updated Author" {
		t.Fatalf("unexpected books_update payload: %+v", updatedBook)
	}

	authorsRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "books_list_authors"})
	if err != nil {
		t.Fatalf("books_list_authors: %v", err)
	}
	authors := decodeToolJSON[struct {
		Authors []struct {
			Author string `json:"author"`
			Count  int    `json:"count"`
		} `json:"authors"`
	}](t, authorsRes)
	if len(authors.Authors) == 0 || authors.Authors[0].Author != "Updated Author" {
		t.Fatalf("unexpected books_list_authors payload: %+v", authors)
	}

	genresRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "books_list_genres"})
	if err != nil {
		t.Fatalf("books_list_genres: %v", err)
	}
	genres := decodeToolJSON[struct {
		Genres []struct {
			Genre string `json:"genre"`
			Count int    `json:"count"`
		} `json:"genres"`
	}](t, genresRes)
	if len(genres.Genres) == 0 || genres.Genres[0].Genre != "Fiction" {
		t.Fatalf("unexpected books_list_genres payload: %+v", genres)
	}

	refreshRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "books_refresh_metadata",
		Arguments: map[string]any{"book_id": "reader-book"},
	})
	if err != nil {
		t.Fatalf("books_refresh_metadata: %v", err)
	}
	refreshed := decodeToolJSON[struct {
		Book domain.Book `json:"book"`
	}](t, refreshRes)
	if refreshed.Book.Title != "Reader Fixture" || refreshed.Book.Author != "Fixture Author" {
		t.Fatalf("unexpected books_refresh_metadata payload: %+v", refreshed)
	}

	createListRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "lists_create",
		Arguments: map[string]any{
			"name":        "Currently Reading",
			"description": "Queue",
		},
	})
	if err != nil {
		t.Fatalf("lists_create: %v", err)
	}
	createdList := decodeToolJSON[struct {
		List domain.BookList `json:"list"`
	}](t, createListRes)
	if createdList.List.Name != "Currently Reading" {
		t.Fatalf("unexpected lists_create payload: %+v", createdList)
	}

	listsRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "lists_list"})
	if err != nil {
		t.Fatalf("lists_list: %v", err)
	}
	lists := decodeToolJSON[struct {
		Lists []domain.BookList `json:"lists"`
	}](t, listsRes)
	if len(lists.Lists) != 1 {
		t.Fatalf("unexpected lists_list payload: %+v", lists)
	}

	updateListRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "lists_update",
		Arguments: map[string]any{
			"list_id":     createdList.List.ID,
			"name":        "Reading Next",
			"description": "Updated queue",
		},
	})
	if err != nil {
		t.Fatalf("lists_update: %v", err)
	}
	updatedList := decodeToolJSON[struct {
		List domain.BookList `json:"list"`
	}](t, updateListRes)
	if updatedList.List.Name != "Reading Next" {
		t.Fatalf("unexpected lists_update payload: %+v", updatedList)
	}

	addRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "lists_add_book",
		Arguments: map[string]any{
			"list_id": createdList.List.ID,
			"book_id": "reader-book",
		},
	})
	if err != nil {
		t.Fatalf("lists_add_book: %v", err)
	}
	added := decodeToolJSON[struct {
		Status string `json:"status"`
		ListID string `json:"list_id"`
		BookID string `json:"book_id"`
	}](t, addRes)
	if added.Status != "added" || added.BookID != "reader-book" {
		t.Fatalf("unexpected lists_add_book payload: %+v", added)
	}

	itemsRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "lists_get_items",
		Arguments: map[string]any{"list_id": createdList.List.ID},
	})
	if err != nil {
		t.Fatalf("lists_get_items: %v", err)
	}
	items := decodeToolJSON[struct {
		Items []domain.BookListItem `json:"items"`
	}](t, itemsRes)
	if len(items.Items) != 1 || items.Items[0].BookID != "reader-book" {
		t.Fatalf("unexpected lists_get_items payload: %+v", items)
	}

	removeRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "lists_remove_book",
		Arguments: map[string]any{
			"list_id": createdList.List.ID,
			"book_id": "reader-book",
		},
	})
	if err != nil {
		t.Fatalf("lists_remove_book: %v", err)
	}
	removed := decodeToolJSON[struct {
		Status string `json:"status"`
	}](t, removeRes)
	if removed.Status != "removed" {
		t.Fatalf("unexpected lists_remove_book payload: %+v", removed)
	}

	manifestRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "reader_get_manifest",
		Arguments: map[string]any{"book_id": "reader-book"},
	})
	if err != nil {
		t.Fatalf("reader_get_manifest: %v", err)
	}
	manifest := decodeToolJSON[struct {
		Manifest domain.ReaderManifest `json:"manifest"`
	}](t, manifestRes)
	if manifest.Manifest.FirstSectionID != "chap-1" {
		t.Fatalf("unexpected reader manifest: %+v", manifest)
	}

	sectionRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "reader_get_section",
		Arguments: map[string]any{
			"book_id":    "reader-book",
			"section_id": "chap-1",
		},
	})
	if err != nil {
		t.Fatalf("reader_get_section: %v", err)
	}
	section := decodeToolJSON[struct {
		Section domain.ReaderSection `json:"section"`
	}](t, sectionRes)
	if !strings.Contains(section.Section.HTML, "/api/v1/books/reader-book/reader/assets/") {
		t.Fatalf("section HTML was not rewritten: %s", section.Section.HTML)
	}

	saveRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "reader_save_progress",
		Arguments: map[string]any{
			"book_id":          "reader-book",
			"section_id":       "chap-1",
			"section_progress": 0.25,
			"block_index":      1,
			"percentage":       0.33,
			"rating":           4,
		},
	})
	if err != nil {
		t.Fatalf("reader_save_progress: %v", err)
	}
	savedProgress := decodeToolJSON[struct {
		Progress domain.ReadingProgress `json:"progress"`
	}](t, saveRes)
	if savedProgress.Progress.SectionID != "chap-1" || savedProgress.Progress.Percentage != 0.33 {
		t.Fatalf("unexpected reader_save_progress payload: %+v", savedProgress)
	}

	progressRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "reader_get_progress",
		Arguments: map[string]any{"book_id": "reader-book"},
	})
	if err != nil {
		t.Fatalf("reader_get_progress: %v", err)
	}
	progress := decodeToolJSON[struct {
		Progress domain.ReadingProgress `json:"progress"`
	}](t, progressRes)
	if progress.Progress.Rating == nil || *progress.Progress.Rating != 4 {
		t.Fatalf("unexpected reader_get_progress payload: %+v", progress)
	}

	toolList, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := make(map[string]bool, len(toolList.Tools))
	for _, tool := range toolList.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"books_search",
		"books_get",
		"books_update",
		"books_refresh_metadata",
		"books_list_authors",
		"books_list_genres",
		"lists_list",
		"lists_create",
		"lists_update",
		"lists_get_items",
		"lists_add_book",
		"lists_remove_book",
		"reader_get_manifest",
		"reader_get_section",
		"reader_get_progress",
		"reader_save_progress",
	} {
		if !names[want] {
			t.Fatalf("missing tool %q in %#v", want, names)
		}
	}
	for _, unwanted := range []string{"books.upload", "books.delete", "books.cover"} {
		if names[unwanted] {
			t.Fatalf("unexpected tool %q exposed in %#v", unwanted, names)
		}
	}

	assertEmptyObjectInputSchema(t, toolList.Tools, "books_list_authors")
	assertEmptyObjectInputSchema(t, toolList.Tools, "books_list_genres")
	assertEmptyObjectInputSchema(t, toolList.Tools, "lists_list")
}

func TestMCPBooksSearchPaginationDefaultsAndCap(t *testing.T) {
	rt := newTestRuntime(t, config.RegistrationModeSingle, "", "")
	registerUser(t, rt, "owner")
	for i := 1; i <= 250; i++ {
		createStoredEPUBBook(t, rt, bookIDForIndex(i), buildReaderFixtureEPUB(t))
	}

	session := connectInMemory(t, rt.server)
	ctx := context.Background()

	for _, tc := range []struct {
		name         string
		args         map[string]any
		wantPage     int
		wantLimit    int
		wantBooksLen int
		wantTotal    int64
	}{
		{
			name:         "omitted limit defaults to one hundred",
			args:         nil,
			wantPage:     1,
			wantLimit:    100,
			wantBooksLen: 100,
			wantTotal:    250,
		},
		{
			name:         "non-positive limit defaults to one hundred",
			args:         map[string]any{"limit": 0},
			wantPage:     1,
			wantLimit:    100,
			wantBooksLen: 100,
			wantTotal:    250,
		},
		{
			name:         "explicit smaller limit is honored",
			args:         map[string]any{"limit": 50},
			wantPage:     1,
			wantLimit:    50,
			wantBooksLen: 50,
			wantTotal:    250,
		},
		{
			name:         "requested limit is clamped to two hundred",
			args:         map[string]any{"limit": 500},
			wantPage:     1,
			wantLimit:    200,
			wantBooksLen: 200,
			wantTotal:    250,
		},
		{
			name:         "non-positive page defaults to one",
			args:         map[string]any{"page": 0, "limit": 50},
			wantPage:     1,
			wantLimit:    50,
			wantBooksLen: 50,
			wantTotal:    250,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
				Name:      "books_search",
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("books_search: %v", err)
			}
			search := decodeToolJSON[struct {
				Books []domain.Book `json:"books"`
				Total int64         `json:"total"`
				Page  int           `json:"page"`
				Limit int           `json:"limit"`
			}](t, res)
			if search.Page != tc.wantPage || search.Limit != tc.wantLimit || len(search.Books) != tc.wantBooksLen || search.Total != tc.wantTotal {
				t.Fatalf("unexpected books_search payload: %+v", search)
			}
		})
	}
}

func TestMCPServerResourcesHappyPath(t *testing.T) {
	rt := newTestRuntime(t, config.RegistrationModeSingle, "", "")
	registerUser(t, rt, "owner")
	createStoredEPUBBook(t, rt, "reader-book", buildReaderFixtureEPUB(t))

	session := connectInMemory(t, rt.server)
	ctx := context.Background()

	createListRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "lists_create",
		Arguments: map[string]any{
			"name": "Shelf",
		},
	})
	if err != nil {
		t.Fatalf("lists_create: %v", err)
	}
	createdList := decodeToolJSON[struct {
		List domain.BookList `json:"list"`
	}](t, createListRes)

	if _, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "lists_add_book",
		Arguments: map[string]any{
			"list_id": createdList.List.ID,
			"book_id": "reader-book",
		},
	}); err != nil {
		t.Fatalf("lists_add_book: %v", err)
	}
	if _, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "reader_save_progress",
		Arguments: map[string]any{
			"book_id":          "reader-book",
			"section_id":       "chap-1",
			"section_progress": 0.5,
			"percentage":       0.5,
		},
	}); err != nil {
		t.Fatalf("reader_save_progress: %v", err)
	}

	templates, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates: %v", err)
	}
	templateNames := make(map[string]bool, len(templates.ResourceTemplates))
	for _, template := range templates.ResourceTemplates {
		templateNames[template.URITemplate] = true
	}
	for _, want := range []string{
		"alexandria://books/{book_id}",
		"alexandria://books/{book_id}/reader/manifest",
		"alexandria://books/{book_id}/reader/sections/{section_id}",
		"alexandria://books/{book_id}/progress",
		"alexandria://lists/{list_id}",
		"alexandria://lists/{list_id}/items",
	} {
		if !templateNames[want] {
			t.Fatalf("missing resource template %q in %#v", want, templateNames)
		}
	}

	bookResource := readResourceJSON[struct {
		Book domain.Book `json:"book"`
	}](t, session, "alexandria://books/reader-book")
	if bookResource.Book.ID != "reader-book" {
		t.Fatalf("unexpected book resource payload: %+v", bookResource)
	}

	manifestResource := readResourceJSON[struct {
		Manifest domain.ReaderManifest `json:"manifest"`
	}](t, session, "alexandria://books/reader-book/reader/manifest")
	if manifestResource.Manifest.FirstSectionID != "chap-1" {
		t.Fatalf("unexpected manifest resource payload: %+v", manifestResource)
	}

	sectionResource := readResourceJSON[struct {
		Section domain.ReaderSection `json:"section"`
	}](t, session, "alexandria://books/reader-book/reader/sections/chap-1")
	if sectionResource.Section.ID != "chap-1" || sectionResource.Section.HTML == "" {
		t.Fatalf("unexpected section resource payload: %+v", sectionResource)
	}

	progressResource := readResourceJSON[struct {
		Progress domain.ReadingProgress `json:"progress"`
	}](t, session, "alexandria://books/reader-book/progress")
	if progressResource.Progress.SectionID != "chap-1" {
		t.Fatalf("unexpected progress resource payload: %+v", progressResource)
	}

	listResource := readResourceJSON[struct {
		List domain.BookList `json:"list"`
	}](t, session, "alexandria://lists/"+createdList.List.ID)
	if listResource.List.ID != createdList.List.ID {
		t.Fatalf("unexpected list resource payload: %+v", listResource)
	}

	itemsResource := readResourceJSON[struct {
		Items []domain.BookListItem `json:"items"`
	}](t, session, "alexandria://lists/"+createdList.List.ID+"/items")
	if len(itemsResource.Items) != 1 || itemsResource.Items[0].BookID != "reader-book" {
		t.Fatalf("unexpected list items resource payload: %+v", itemsResource)
	}
}

func TestMCPServerOwnerResolutionRequiresConfigForUserScopedOps(t *testing.T) {
	rt := newTestRuntime(t, config.RegistrationModeMulti, "", "")
	registerUser(t, rt, "owner-one")
	registerUser(t, rt, "owner-two")
	createStoredEPUBBook(t, rt, "reader-book", buildReaderFixtureEPUB(t))

	session := connectInMemory(t, rt.server)
	ctx := context.Background()

	listsRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: "lists_list"})
	if err != nil {
		t.Fatalf("lists_list: %v", err)
	}
	if !listsRes.IsError || !strings.Contains(toolResultText(t, listsRes), "MCP_OWNER_USERNAME") {
		t.Fatalf("expected actionable owner resolution error, got %+v", listsRes)
	}

	getRes, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "books_get",
		Arguments: map[string]any{"book_id": "reader-book"},
	})
	if err != nil {
		t.Fatalf("books_get: %v", err)
	}
	if getRes.IsError {
		t.Fatalf("expected non-user-scoped tool to succeed, got %+v", getRes)
	}
}

func TestMCPHTTPRouteAuthAndRegistration(t *testing.T) {
	t.Run("disabled without token", func(t *testing.T) {
		rt := newTestRuntime(t, config.RegistrationModeSingle, "", "")
		if handler := rt.server.AuthenticatedHTTPHandler(); handler != nil {
			t.Fatal("expected nil HTTP handler when MCP_HTTP_TOKEN is unset")
		}
	})

	t.Run("requires valid bearer token", func(t *testing.T) {
		rt := newTestRuntime(t, config.RegistrationModeSingle, "mcp-secret", "")
		handler := rt.server.AuthenticatedHTTPHandler()
		if handler == nil {
			t.Fatal("expected authenticated HTTP handler")
		}

		for _, tc := range []struct {
			name  string
			token string
		}{
			{name: "missing", token: ""},
			{name: "invalid", token: "wrong-secret"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "http://alexandria.test/mcp", strings.NewReader(`{}`))
				req.Header.Set("Content-Type", "application/json")
				if tc.token != "" {
					req.Header.Set("Authorization", "Bearer "+tc.token)
				}

				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, req)
				if resp.Code != http.StatusUnauthorized {
					t.Fatalf("expected 401 for %s token, got %d", tc.name, resp.Code)
				}
			})
		}

		client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "mcp-test-client", Version: "1.0.0"}, nil)
		session, err := client.Connect(context.Background(), &sdkmcp.StreamableClientTransport{
			Endpoint:             "http://alexandria.test/mcp",
			DisableStandaloneSSE: true,
			HTTPClient: &http.Client{
				Transport: &authRoundTripper{
					base:  &handlerRoundTripper{handler: handler},
					token: "mcp-secret",
				},
			},
		}, nil)
		if err != nil {
			t.Fatalf("client.Connect over HTTP: %v", err)
		}
		defer session.Close()

		tools, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListTools over HTTP: %v", err)
		}
		if len(tools.Tools) == 0 {
			t.Fatal("expected MCP HTTP route to expose tools")
		}
	})
}

func TestMCPStdioLikeIOTransportInitialization(t *testing.T) {
	rt := newTestRuntime(t, config.RegistrationModeSingle, "", "")

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	serverTransport := &sdkmcp.IOTransport{Reader: serverReader, Writer: serverWriter}
	clientTransport := &sdkmcp.IOTransport{Reader: clientReader, Writer: clientWriter}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- rt.server.Run(ctx, serverTransport)
	}()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "stdio-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect over IOTransport: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools over IOTransport: %v", err)
	}
	if len(tools.Tools) == 0 {
		t.Fatal("expected tools over IOTransport")
	}

	session.Close()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("server.Run returned unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("timed out waiting for IOTransport server to exit")
	}
}

func newTestRuntime(t *testing.T, mode config.RegistrationMode, httpToken, ownerUsername string) *testRuntime {
	t.Helper()

	baseDir := t.TempDir()
	cfg := &config.Config{
		Port:             "0",
		DataDir:          baseDir + "/data",
		DBPath:           baseDir + "/data/alexandria.db",
		JWTSecret:        "test-secret-with-sufficient-length-123456",
		JWTExpiry:        15 * time.Minute,
		RefreshExpiry:    24 * time.Hour,
		UploadMaxBytes:   8 * 1024 * 1024,
		RegistrationMode: mode,
		MCPHTTPToken:     httpToken,
		MCPOwnerUsername: ownerUsername,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	bookParser := parser.New(map[domain.FileType]ports.BookParser{
		domain.FileTypeEPUB: epub.New(),
		domain.FileTypePDF:  pdf.New(),
	})

	application := app.New(
		sqlite.NewUserRepo(db),
		sqlite.NewBookRepo(db),
		sqlite.NewProgressRepo(db),
		sqlite.NewListRepo(db),
		storage.NewLocalFileStore(cfg.DataDir),
		bookParser,
		nil,
		cfg,
	)

	return &testRuntime{
		cfg:         cfg,
		db:          db,
		application: application,
		server:      mcpserver.New(application, cfg),
	}
}

func registerUser(t *testing.T, rt *testRuntime, username string) {
	t.Helper()
	if _, _, err := rt.application.Auth.Register(context.Background(), username, "testpass123"); err != nil {
		t.Fatalf("register user %q: %v", username, err)
	}
}

func connectInMemory(t *testing.T, server *mcpserver.Server) *sdkmcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()

	serverSession, err := server.Protocol().Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "mcp-test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		serverSession.Close()
		t.Fatalf("client.Connect: %v", err)
	}

	t.Cleanup(func() {
		clientSession.Close()
		serverSession.Close()
	})

	return clientSession
}

func decodeToolJSON[T any](t *testing.T, res *sdkmcp.CallToolResult) T {
	t.Helper()
	if res.IsError {
		t.Fatalf("expected non-error tool result, got %q", toolResultText(t, res))
	}
	return decodeJSON[T](t, toolResultText(t, res))
}

func readResourceJSON[T any](t *testing.T, session *sdkmcp.ClientSession, uri string) T {
	t.Helper()
	res, err := session.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource(%q): %v", uri, err)
	}
	if len(res.Contents) != 1 {
		t.Fatalf("expected one resource content for %q, got %d", uri, len(res.Contents))
	}
	return decodeJSON[T](t, res.Contents[0].Text)
}

func assertEmptyObjectInputSchema(t *testing.T, tools []*sdkmcp.Tool, toolName string) {
	t.Helper()

	for _, tool := range tools {
		if tool.Name != toolName {
			continue
		}

		schema, ok := tool.InputSchema.(map[string]any)
		if !ok {
			t.Fatalf("tool %q input schema has unexpected type %T", toolName, tool.InputSchema)
		}
		if schema["type"] != "object" {
			t.Fatalf("tool %q input schema type = %#v, want object", toolName, schema["type"])
		}

		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("tool %q input schema missing object properties: %#v", toolName, schema)
		}
		if len(properties) != 0 {
			t.Fatalf("tool %q input schema properties = %#v, want empty object", toolName, properties)
		}
		return
	}

	t.Fatalf("tool %q not found while checking input schema", toolName)
}

func bookIDForIndex(i int) string {
	return fmt.Sprintf("reader-book-%03d", i)
}

func decodeJSON[T any](t *testing.T, text string) T {
	t.Helper()
	var value T
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		t.Fatalf("decode JSON %q: %v", text, err)
	}
	return value
}

func toolResultText(t *testing.T, res *sdkmcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("tool result did not include content")
	}
	text, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("tool result content was %T, want *TextContent", res.Content[0])
	}
	return text.Text
}

type authRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+rt.token)
	return rt.base.RoundTrip(clone)
}

type handlerRoundTripper struct {
	handler http.Handler
}

func (rt *handlerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	clone := req.Clone(req.Context())
	clone.RequestURI = ""
	rt.handler.ServeHTTP(recorder, clone)
	return recorder.Result(), nil
}

func createStoredEPUBBook(t *testing.T, rt *testRuntime, bookID string, payload []byte) {
	t.Helper()

	filePath := filepath.Join(rt.cfg.DataDir, "books", bookID, "original.epub")
	if err := createDirForFile(filePath); err != nil {
		t.Fatalf("create book dir: %v", err)
	}
	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		t.Fatalf("write EPUB fixture: %v", err)
	}

	book := &domain.Book{
		ID:        bookID,
		Title:     "Fixture Book",
		Author:    "Fixture Author",
		FilePath:  filepath.Join("books", bookID, "original.epub"),
		FileType:  domain.FileTypeEPUB,
		FileSize:  int64(len(payload)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sqlite.NewBookRepo(rt.db).Create(context.Background(), book); err != nil {
		t.Fatalf("create book fixture: %v", err)
	}
}

func createDirForFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func buildReaderFixtureEPUB(t *testing.T) []byte {
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
		t.Fatalf("close EPUB writer: %v", err)
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
