package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"alexandria/internal/app"
	apprepos "alexandria/internal/app/repos"
	"alexandria/internal/app/services"
	"alexandria/internal/config"
	"alexandria/internal/domain"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	resourceScheme         = "alexandria"
	defaultBooksSearchLimit = 100
	maxBooksSearchLimit     = 200
)

var (
	genericObjectSchema = map[string]any{"type": "object"}
	emptyInputSchema    = map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
)

// Server wraps the shared MCP registration and transport helpers for Alexandria.
type Server struct {
	protocol *sdkmcp.Server

	books  *services.BookService
	reader *services.ReaderService
	lists  *services.ListService
	users  *services.UserService

	httpToken     string
	ownerUsername string
}

func New(application *app.App, cfg *config.Config) *Server {
	s := &Server{
		books:         application.Books,
		reader:        application.Reader,
		lists:         application.Lists,
		users:         application.Users,
		httpToken:     cfg.MCPHTTPToken,
		ownerUsername: cfg.MCPOwnerUsername,
	}

	s.protocol = sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "alexandria",
		Version: "0.1.0",
	}, nil)

	s.registerTools()
	s.registerResources()
	return s
}

func (s *Server) Protocol() *sdkmcp.Server {
	return s.protocol
}

func (s *Server) Run(ctx context.Context, transport sdkmcp.Transport) error {
	return s.protocol.Run(ctx, transport)
}

func (s *Server) AuthenticatedHTTPHandler() http.Handler {
	if strings.TrimSpace(s.httpToken) == "" {
		return nil
	}

	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return s.protocol
	}, nil)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !bearerTokenMatches(req.Header.Get("Authorization"), s.httpToken) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="alexandria-mcp"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, req)
	})
}

type emptyInput struct{}

type booksSearchInput struct {
	Search    string `json:"search,omitempty" jsonschema:"Optional search text for title, author, or description"`
	Author    string `json:"author,omitempty" jsonschema:"Optional author substring filter"`
	Genre     string `json:"genre,omitempty" jsonschema:"Optional exact genre filter"`
	SortBy    string `json:"sort_by,omitempty" jsonschema:"Sort by one of: title, author, created_at, rating, file_size"`
	SortOrder string `json:"sort_order,omitempty" jsonschema:"Sort order: asc or desc"`
	Page      int    `json:"page,omitempty" jsonschema:"1-based page number; defaults to 1"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Page size; defaults to 100 and is capped at 200"`
}

type bookIdentifierInput struct {
	BookID string `json:"book_id" jsonschema:"Alexandria book ID"`
}

type bookUpdateMetadataInput struct {
	ISBN        *string   `json:"isbn,omitempty" jsonschema:"ISBN value"`
	Publisher   *string   `json:"publisher,omitempty" jsonschema:"Publisher name"`
	PublishedAt *string   `json:"published_at,omitempty" jsonschema:"Publication date in YYYY-MM-DD or RFC3339 format"`
	Language    *string   `json:"language,omitempty" jsonschema:"Language code or label"`
	Genres      *[]string `json:"genres,omitempty" jsonschema:"Full replacement genre list"`
	Tags        *[]string `json:"tags,omitempty" jsonschema:"Full replacement tag list"`
}

type booksUpdateInput struct {
	BookID          string                   `json:"book_id" jsonschema:"Alexandria book ID"`
	Title           *string                  `json:"title,omitempty" jsonschema:"Optional replacement title"`
	Author          *string                  `json:"author,omitempty" jsonschema:"Optional replacement author"`
	Description     *string                  `json:"description,omitempty" jsonschema:"Optional replacement description"`
	ZealotTicketID  *string                  `json:"zealot_ticket_id,omitempty" jsonschema:"Optional linked Zealot ticket ID"`
	MetadataUpdates *bookUpdateMetadataInput `json:"metadata,omitempty" jsonschema:"Optional bibliographic metadata updates"`
}

type listIdentifierInput struct {
	ListID string `json:"list_id" jsonschema:"Alexandria list ID"`
}

type listMutationInput struct {
	ListID string `json:"list_id" jsonschema:"Alexandria list ID"`
	BookID string `json:"book_id" jsonschema:"Alexandria book ID"`
}

type listCreateInput struct {
	Name        string `json:"name" jsonschema:"List name"`
	Description string `json:"description,omitempty" jsonschema:"Optional list description"`
}

type listUpdateInput struct {
	ListID      string  `json:"list_id" jsonschema:"Alexandria list ID"`
	Name        *string `json:"name,omitempty" jsonschema:"Optional replacement list name"`
	Description *string `json:"description,omitempty" jsonschema:"Optional replacement description"`
}

type readerSectionInput struct {
	BookID    string `json:"book_id" jsonschema:"Alexandria book ID"`
	SectionID string `json:"section_id" jsonschema:"Reader section ID"`
}

type readerSaveProgressInput struct {
	BookID          string  `json:"book_id" jsonschema:"Alexandria book ID"`
	SectionID       string  `json:"section_id" jsonschema:"Reader section ID"`
	SectionProgress float64 `json:"section_progress" jsonschema:"Scroll progress within the current section, from 0 to 1"`
	BlockIndex      *int    `json:"block_index,omitempty" jsonschema:"Optional readable block index"`
	Percentage      float64 `json:"percentage" jsonschema:"Overall book progress, from 0 to 1"`
	Rating          *int    `json:"rating,omitempty" jsonschema:"Optional 1-5 rating"`
}

type booksSearchOutput struct {
	Books []*domain.Book `json:"books"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

type bookOutput struct {
	Book *domain.Book `json:"book"`
}

type authorsOutput struct {
	Authors []apprepos.AuthorSummary `json:"authors"`
}

type genresOutput struct {
	Genres []apprepos.GenreSummary `json:"genres"`
}

type listsOutput struct {
	Lists []*domain.BookList `json:"lists"`
}

type listOutput struct {
	List *domain.BookList `json:"list"`
}

type listItemsOutput struct {
	Items []*domain.BookListItem `json:"items"`
}

type listMutationOutput struct {
	Status string `json:"status"`
	ListID string `json:"list_id"`
	BookID string `json:"book_id"`
}

type manifestOutput struct {
	Manifest *domain.ReaderManifest `json:"manifest"`
}

type sectionOutput struct {
	Section *domain.ReaderSection `json:"section"`
}

type progressOutput struct {
	Progress *domain.ReadingProgress `json:"progress"`
}

func (s *Server) registerTools() {
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "books_search",
		Description: "Search Alexandria books with optional filtering, sorting, and pagination.",
	}, s.handleBooksSearch)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "books_get",
		Description: "Fetch one Alexandria book by ID.",
	}, s.handleBooksGet)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "books_update",
		Description: "Update safe Alexandria book metadata fields.",
	}, s.handleBooksUpdate)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "books_refresh_metadata",
		Description: "Re-extract metadata and cover information from a stored Alexandria book file.",
	}, s.handleBooksRefreshMetadata)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "books_list_authors",
		Description: "List distinct Alexandria authors with book counts.",
		InputSchema: emptyInputSchema,
	}, s.handleBooksListAuthors)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "books_list_genres",
		Description: "List distinct Alexandria genres with book counts.",
		InputSchema: emptyInputSchema,
	}, s.handleBooksListGenres)

	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "lists_list",
		Description: "List Alexandria lists for the configured MCP owner.",
		InputSchema: emptyInputSchema,
	}, s.handleListsList)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "lists_create",
		Description: "Create an Alexandria list for the configured MCP owner.",
	}, s.handleListsCreate)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "lists_update",
		Description: "Update an Alexandria list for the configured MCP owner.",
	}, s.handleListsUpdate)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "lists_get_items",
		Description: "List books contained in an Alexandria list for the configured MCP owner.",
	}, s.handleListsGetItems)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "lists_add_book",
		Description: "Add a book to an Alexandria list for the configured MCP owner.",
	}, s.handleListsAddBook)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "lists_remove_book",
		Description: "Remove a book from an Alexandria list for the configured MCP owner.",
	}, s.handleListsRemoveBook)

	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:         "reader_get_manifest",
		Description:  "Fetch the Alexandria reader manifest for an EPUB book.",
		OutputSchema: genericObjectSchema,
	}, s.handleReaderGetManifest)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "reader_get_section",
		Description: "Fetch one Alexandria reader section, including rewritten HTML.",
	}, s.handleReaderGetSection)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "reader_get_progress",
		Description: "Fetch saved reading progress for the configured MCP owner.",
	}, s.handleReaderGetProgress)
	sdkmcp.AddTool(s.protocol, &sdkmcp.Tool{
		Name:        "reader_save_progress",
		Description: "Save reading progress for the configured MCP owner.",
	}, s.handleReaderSaveProgress)
}

func (s *Server) registerResources() {
	s.protocol.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		Name:        "book",
		Title:       "Alexandria Book",
		Description: "Read one Alexandria book record as JSON.",
		MIMEType:    "application/json",
		URITemplate: "alexandria://books/{book_id}",
	}, s.readBookResource)
	s.protocol.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		Name:        "reader_manifest",
		Title:       "Alexandria Reader Manifest",
		Description: "Read the Alexandria reader manifest for an EPUB book as JSON.",
		MIMEType:    "application/json",
		URITemplate: "alexandria://books/{book_id}/reader/manifest",
	}, s.readManifestResource)
	s.protocol.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		Name:        "reader_section",
		Title:       "Alexandria Reader Section",
		Description: "Read one Alexandria reader section, including rewritten HTML, as JSON.",
		MIMEType:    "application/json",
		URITemplate: "alexandria://books/{book_id}/reader/sections/{section_id}",
	}, s.readSectionResource)
	s.protocol.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		Name:        "reader_progress",
		Title:       "Alexandria Reading Progress",
		Description: "Read saved reading progress for the configured MCP owner as JSON.",
		MIMEType:    "application/json",
		URITemplate: "alexandria://books/{book_id}/progress",
	}, s.readProgressResource)
	s.protocol.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		Name:        "list",
		Title:       "Alexandria List",
		Description: "Read one Alexandria list for the configured MCP owner as JSON.",
		MIMEType:    "application/json",
		URITemplate: "alexandria://lists/{list_id}",
	}, s.readListResource)
	s.protocol.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		Name:        "list_items",
		Title:       "Alexandria List Items",
		Description: "Read the books in an Alexandria list for the configured MCP owner as JSON.",
		MIMEType:    "application/json",
		URITemplate: "alexandria://lists/{list_id}/items",
	}, s.readListItemsResource)
}

func (s *Server) handleBooksSearch(ctx context.Context, _ *sdkmcp.CallToolRequest, input booksSearchInput) (*sdkmcp.CallToolResult, booksSearchOutput, error) {
	page, limit := normalizePagination(input.Page, input.Limit)
	filter := apprepos.BookFilter{
		Search:    input.Search,
		Author:    input.Author,
		Genre:     input.Genre,
		SortBy:    input.SortBy,
		SortOrder: input.SortOrder,
		Page:      page,
		Limit:     limit,
	}
	if strings.EqualFold(input.SortBy, "rating") {
		userID, err := s.ownerUserID(ctx)
		if err != nil {
			return nil, booksSearchOutput{}, err
		}
		filter.UserID = userID
	}

	books, total, err := s.books.List(ctx, filter)
	if err != nil {
		return nil, booksSearchOutput{}, err
	}
	return nil, booksSearchOutput{
		Books: books,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *Server) handleBooksGet(ctx context.Context, _ *sdkmcp.CallToolRequest, input bookIdentifierInput) (*sdkmcp.CallToolResult, bookOutput, error) {
	book, err := s.books.GetByID(ctx, input.BookID)
	if err != nil {
		return nil, bookOutput{}, err
	}
	return nil, bookOutput{Book: book}, nil
}

func (s *Server) handleBooksUpdate(ctx context.Context, _ *sdkmcp.CallToolRequest, input booksUpdateInput) (*sdkmcp.CallToolResult, bookOutput, error) {
	update := services.BookUpdate{
		Title:          input.Title,
		Author:         input.Author,
		Description:    input.Description,
		ZealotTicketID: input.ZealotTicketID,
	}
	if input.MetadataUpdates != nil {
		update.ISBN = input.MetadataUpdates.ISBN
		update.Publisher = input.MetadataUpdates.Publisher
		update.Language = input.MetadataUpdates.Language
		update.Genres = input.MetadataUpdates.Genres
		update.Tags = input.MetadataUpdates.Tags
		if input.MetadataUpdates.PublishedAt != nil && strings.TrimSpace(*input.MetadataUpdates.PublishedAt) != "" {
			parsed, err := parseDateInput(*input.MetadataUpdates.PublishedAt)
			if err != nil {
				return nil, bookOutput{}, fmt.Errorf("published_at must be YYYY-MM-DD or RFC3339")
			}
			update.PublishedAt = &parsed
		}
	}

	book, err := s.books.Update(ctx, input.BookID, update)
	if err != nil {
		return nil, bookOutput{}, err
	}
	return nil, bookOutput{Book: book}, nil
}

func (s *Server) handleBooksRefreshMetadata(ctx context.Context, _ *sdkmcp.CallToolRequest, input bookIdentifierInput) (*sdkmcp.CallToolResult, bookOutput, error) {
	book, err := s.books.RefreshMetadata(ctx, input.BookID)
	if err != nil {
		return nil, bookOutput{}, err
	}
	return nil, bookOutput{Book: book}, nil
}

func (s *Server) handleBooksListAuthors(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, authorsOutput, error) {
	authors, err := s.books.ListAuthors(ctx)
	if err != nil {
		return nil, authorsOutput{}, err
	}
	return nil, authorsOutput{Authors: authors}, nil
}

func (s *Server) handleBooksListGenres(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, genresOutput, error) {
	genres, err := s.books.ListGenres(ctx)
	if err != nil {
		return nil, genresOutput{}, err
	}
	return nil, genresOutput{Genres: genres}, nil
}

func (s *Server) handleListsList(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyInput) (*sdkmcp.CallToolResult, listsOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, listsOutput{}, err
	}
	lists, err := s.lists.ListLists(ctx, userID)
	if err != nil {
		return nil, listsOutput{}, err
	}
	return nil, listsOutput{Lists: lists}, nil
}

func (s *Server) handleListsCreate(ctx context.Context, _ *sdkmcp.CallToolRequest, input listCreateInput) (*sdkmcp.CallToolResult, listOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, listOutput{}, err
	}
	list, err := s.lists.CreateList(ctx, userID, input.Name, input.Description)
	if err != nil {
		return nil, listOutput{}, err
	}
	return nil, listOutput{List: list}, nil
}

func (s *Server) handleListsUpdate(ctx context.Context, _ *sdkmcp.CallToolRequest, input listUpdateInput) (*sdkmcp.CallToolResult, listOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, listOutput{}, err
	}
	list, err := s.lists.UpdateList(ctx, userID, input.ListID, input.Name, input.Description)
	if err != nil {
		return nil, listOutput{}, err
	}
	return nil, listOutput{List: list}, nil
}

func (s *Server) handleListsGetItems(ctx context.Context, _ *sdkmcp.CallToolRequest, input listIdentifierInput) (*sdkmcp.CallToolResult, listItemsOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, listItemsOutput{}, err
	}
	items, err := s.lists.GetItems(ctx, userID, input.ListID)
	if err != nil {
		return nil, listItemsOutput{}, err
	}
	return nil, listItemsOutput{Items: items}, nil
}

func (s *Server) handleListsAddBook(ctx context.Context, _ *sdkmcp.CallToolRequest, input listMutationInput) (*sdkmcp.CallToolResult, listMutationOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, listMutationOutput{}, err
	}
	if err := s.lists.AddBook(ctx, userID, input.ListID, input.BookID); err != nil {
		return nil, listMutationOutput{}, err
	}
	return nil, listMutationOutput{Status: "added", ListID: input.ListID, BookID: input.BookID}, nil
}

func (s *Server) handleListsRemoveBook(ctx context.Context, _ *sdkmcp.CallToolRequest, input listMutationInput) (*sdkmcp.CallToolResult, listMutationOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, listMutationOutput{}, err
	}
	if err := s.lists.RemoveBook(ctx, userID, input.ListID, input.BookID); err != nil {
		return nil, listMutationOutput{}, err
	}
	return nil, listMutationOutput{Status: "removed", ListID: input.ListID, BookID: input.BookID}, nil
}

func (s *Server) handleReaderGetManifest(ctx context.Context, _ *sdkmcp.CallToolRequest, input bookIdentifierInput) (*sdkmcp.CallToolResult, manifestOutput, error) {
	manifest, err := s.reader.GetManifest(ctx, input.BookID)
	if err != nil {
		return nil, manifestOutput{}, err
	}
	return nil, manifestOutput{Manifest: manifest}, nil
}

func (s *Server) handleReaderGetSection(ctx context.Context, _ *sdkmcp.CallToolRequest, input readerSectionInput) (*sdkmcp.CallToolResult, sectionOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, sectionOutput{}, err
	}
	section, err := s.reader.GetSection(ctx, userID, input.BookID, input.SectionID)
	if err != nil {
		return nil, sectionOutput{}, err
	}
	return nil, sectionOutput{Section: section}, nil
}

func (s *Server) handleReaderGetProgress(ctx context.Context, _ *sdkmcp.CallToolRequest, input bookIdentifierInput) (*sdkmcp.CallToolResult, progressOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, progressOutput{}, err
	}
	progress, err := s.reader.GetProgress(ctx, userID, input.BookID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, progressOutput{Progress: nil}, nil
	}
	if err != nil {
		return nil, progressOutput{}, err
	}
	return nil, progressOutput{Progress: progress}, nil
}

func (s *Server) handleReaderSaveProgress(ctx context.Context, _ *sdkmcp.CallToolRequest, input readerSaveProgressInput) (*sdkmcp.CallToolResult, progressOutput, error) {
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, progressOutput{}, err
	}
	progress, err := s.reader.SaveProgress(
		ctx,
		userID,
		input.BookID,
		input.SectionID,
		input.SectionProgress,
		input.BlockIndex,
		input.Percentage,
		input.Rating,
	)
	if err != nil {
		return nil, progressOutput{}, err
	}
	return nil, progressOutput{Progress: progress}, nil
}

func (s *Server) readBookResource(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	bookID, err := parseBookResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	book, err := s.books.GetByID(ctx, bookID)
	if err != nil {
		return nil, resourceError(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, bookOutput{Book: book})
}

func (s *Server) readManifestResource(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	bookID, err := parseManifestResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	manifest, err := s.reader.GetManifest(ctx, bookID)
	if err != nil {
		return nil, resourceError(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, manifestOutput{Manifest: manifest})
}

func (s *Server) readSectionResource(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	bookID, sectionID, err := parseSectionResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, err
	}
	section, err := s.reader.GetSection(ctx, userID, bookID, sectionID)
	if err != nil {
		return nil, resourceError(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, sectionOutput{Section: section})
}

func (s *Server) readProgressResource(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	bookID, err := parseProgressResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, err
	}
	progress, err := s.reader.GetProgress(ctx, userID, bookID)
	if errors.Is(err, domain.ErrNotFound) {
		return jsonResource(req.Params.URI, progressOutput{Progress: nil})
	}
	if err != nil {
		return nil, resourceError(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, progressOutput{Progress: progress})
}

func (s *Server) readListResource(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	listID, err := parseListResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.lists.GetList(ctx, userID, listID)
	if err != nil {
		return nil, resourceError(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, listOutput{List: list})
}

func (s *Server) readListItemsResource(ctx context.Context, req *sdkmcp.ReadResourceRequest) (*sdkmcp.ReadResourceResult, error) {
	listID, err := parseListItemsResourceURI(req.Params.URI)
	if err != nil {
		return nil, err
	}
	userID, err := s.ownerUserID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.lists.GetItems(ctx, userID, listID)
	if err != nil {
		return nil, resourceError(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, listItemsOutput{Items: items})
}

func (s *Server) ownerUserID(ctx context.Context) (string, error) {
	user, err := s.ownerUser(ctx)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (s *Server) ownerUser(ctx context.Context) (*domain.User, error) {
	if s.ownerUsername != "" {
		user, err := s.users.GetByUsername(ctx, s.ownerUsername)
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("configured MCP owner %q was not found", s.ownerUsername)
		}
		if err != nil {
			return nil, err
		}
		return user, nil
	}

	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}
	switch len(users) {
	case 0:
		return nil, fmt.Errorf("user-scoped MCP operations require an Alexandria account; create a user first")
	case 1:
		return users[0], nil
	default:
		return nil, fmt.Errorf("MCP_OWNER_USERNAME is required for user-scoped MCP operations when multiple Alexandria users exist")
	}
}

func normalizePagination(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = defaultBooksSearchLimit
	}
	if limit > maxBooksSearchLimit {
		limit = maxBooksSearchLimit
	}
	return page, limit
}

func parseDateInput(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if parsed, err := time.Parse(layout, s); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q", s)
}

func bearerTokenMatches(header, want string) bool {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	return strings.TrimSpace(header[len(prefix):]) == want
}

func jsonResource(uri string, payload any) (*sdkmcp.ReadResourceResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal resource %q: %w", uri, err)
	}
	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(body),
		}},
	}, nil
}

func resourceError(uri string, err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return sdkmcp.ResourceNotFoundError(uri)
	}
	return err
}

func parseBookResourceURI(raw string) (string, error) {
	host, segments, err := parseResourceURI(raw)
	if err != nil || host != "books" || len(segments) != 1 {
		return "", sdkmcp.ResourceNotFoundError(raw)
	}
	return segments[0], nil
}

func parseManifestResourceURI(raw string) (string, error) {
	host, segments, err := parseResourceURI(raw)
	if err != nil || host != "books" || len(segments) != 3 || segments[1] != "reader" || segments[2] != "manifest" {
		return "", sdkmcp.ResourceNotFoundError(raw)
	}
	return segments[0], nil
}

func parseSectionResourceURI(raw string) (string, string, error) {
	host, segments, err := parseResourceURI(raw)
	if err != nil || host != "books" || len(segments) != 4 || segments[1] != "reader" || segments[2] != "sections" {
		return "", "", sdkmcp.ResourceNotFoundError(raw)
	}
	return segments[0], segments[3], nil
}

func parseProgressResourceURI(raw string) (string, error) {
	host, segments, err := parseResourceURI(raw)
	if err != nil || host != "books" || len(segments) != 2 || segments[1] != "progress" {
		return "", sdkmcp.ResourceNotFoundError(raw)
	}
	return segments[0], nil
}

func parseListResourceURI(raw string) (string, error) {
	host, segments, err := parseResourceURI(raw)
	if err != nil || host != "lists" || len(segments) != 1 {
		return "", sdkmcp.ResourceNotFoundError(raw)
	}
	return segments[0], nil
}

func parseListItemsResourceURI(raw string) (string, error) {
	host, segments, err := parseResourceURI(raw)
	if err != nil || host != "lists" || len(segments) != 2 || segments[1] != "items" {
		return "", sdkmcp.ResourceNotFoundError(raw)
	}
	return segments[0], nil
}

func parseResourceURI(raw string) (string, []string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", nil, err
	}
	if parsed.Scheme != resourceScheme {
		return "", nil, fmt.Errorf("unsupported resource scheme %q", parsed.Scheme)
	}

	trimmedPath := strings.Trim(parsed.Path, "/")
	if trimmedPath == "" {
		return parsed.Host, nil, nil
	}
	return parsed.Host, strings.Split(trimmedPath, "/"), nil
}
