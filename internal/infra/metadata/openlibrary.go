package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"alexandria/internal/app/ports"
)

// OpenLibraryProvider queries the Open Library search API (no API key required).
type OpenLibraryProvider struct {
	client *http.Client
}

func NewOpenLibrary() *OpenLibraryProvider {
	return &OpenLibraryProvider{client: &http.Client{}}
}

func (p *OpenLibraryProvider) ID() string { return "open_library" }

func (p *OpenLibraryProvider) Search(ctx context.Context, q ports.MetadataQuery) ([]ports.MetadataResult, error) {
	if q.ISBN != "" {
		return p.searchByISBN(ctx, q.ISBN)
	}
	return p.searchByQuery(ctx, q)
}

func (p *OpenLibraryProvider) searchByISBN(ctx context.Context, isbn string) ([]ports.MetadataResult, error) {
	reqURL := "https://openlibrary.org/isbn/" + url.PathEscape(isbn) + ".json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("open library: build isbn request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open library: isbn request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open library: HTTP %d", resp.StatusCode)
	}

	var doc olISBNDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("open library: decode isbn: %w", err)
	}

	result := olISBNDocToResult(isbn, doc)
	return []ports.MetadataResult{result}, nil
}

func (p *OpenLibraryProvider) searchByQuery(ctx context.Context, q ports.MetadataQuery) ([]ports.MetadataResult, error) {
	params := url.Values{}
	params.Set("limit", "10")
	params.Set("fields", "key,title,author_name,first_publish_year,isbn,publisher,language,subject,cover_i")

	if q.Title != "" {
		params.Set("title", q.Title)
	}
	if q.Author != "" {
		params.Set("author", q.Author)
	}
	if q.Q != "" && q.Title == "" && q.Author == "" {
		params.Set("q", q.Q)
	}

	reqURL := "https://openlibrary.org/search.json?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("open library: build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open library: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open library: HTTP %d", resp.StatusCode)
	}

	var payload olSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("open library: decode: %w", err)
	}

	results := make([]ports.MetadataResult, 0, len(payload.Docs))
	for _, doc := range payload.Docs {
		results = append(results, olDocToResult(doc))
	}
	return results, nil
}

// --- conversion helpers ---

func olDocToResult(doc olSearchDoc) ports.MetadataResult {
	var isbn string
	if len(doc.ISBN) > 0 {
		isbn = doc.ISBN[0]
	}

	var publisher string
	if len(doc.Publisher) > 0 {
		publisher = doc.Publisher[0]
	}

	var lang string
	if len(doc.Language) > 0 {
		lang = doc.Language[0]
	}

	var coverURL string
	if doc.CoverI != 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", doc.CoverI)
	}

	var publishedDate string
	if doc.FirstPublishYear != 0 {
		publishedDate = strconv.Itoa(doc.FirstPublishYear)
	}

	tags := doc.Subject
	if len(tags) > 10 {
		tags = tags[:10]
	}

	externalURL := ""
	if doc.Key != "" {
		externalURL = "https://openlibrary.org" + doc.Key
	}

	return ports.MetadataResult{
		Title:         doc.Title,
		Authors:       doc.AuthorName,
		CoverURL:      coverURL,
		Publisher:     publisher,
		PublishedDate: publishedDate,
		ISBN:          isbn,
		Language:      lang,
		Tags:          tags,
		ExternalURL:   externalURL,
		Source: ports.MetadataSource{
			ID:   "open_library",
			Name: "Open Library",
			Link: "https://openlibrary.org",
		},
	}
}

func olISBNDocToResult(isbn string, doc olISBNDoc) ports.MetadataResult {
	var authors []string
	for _, a := range doc.Authors {
		if a.Key != "" {
			// Author key only — name resolution would need another call; skip for simplicity
			authors = append(authors, a.Key)
		}
	}

	var publisher string
	if len(doc.Publishers) > 0 {
		publisher = doc.Publishers[0]
	}

	var coverURL string
	if doc.Covers != nil && len(doc.Covers) > 0 {
		coverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", doc.Covers[0])
	}

	externalURL := ""
	if doc.Key != "" {
		externalURL = "https://openlibrary.org" + doc.Key
	}

	return ports.MetadataResult{
		Title:         doc.Title,
		Authors:       authors,
		CoverURL:      coverURL,
		Publisher:     publisher,
		PublishedDate: doc.PublishDate,
		ISBN:          isbn,
		ExternalURL:   externalURL,
		Source: ports.MetadataSource{
			ID:   "open_library",
			Name: "Open Library",
			Link: "https://openlibrary.org",
		},
	}
}

// --- JSON structs ---

type olSearchResponse struct {
	Docs []olSearchDoc `json:"docs"`
}

type olSearchDoc struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	ISBN             []string `json:"isbn"`
	Publisher        []string `json:"publisher"`
	Language         []string `json:"language"`
	Subject          []string `json:"subject"`
	CoverI           int64    `json:"cover_i"`
}

type olISBNDoc struct {
	Key         string          `json:"key"`
	Title       string          `json:"title"`
	Authors     []olAuthorRef   `json:"authors"`
	Publishers  []string        `json:"publishers"`
	PublishDate string          `json:"publish_date"`
	Covers      []int64         `json:"covers"`
}

type olAuthorRef struct {
	Key string `json:"key"`
}
