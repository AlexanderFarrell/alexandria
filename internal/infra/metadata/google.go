package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"alexandria/internal/app/ports"
)

// GoogleBooksProvider queries the Google Books public API (no API key required).
type GoogleBooksProvider struct {
	client *http.Client
}

func NewGoogleBooks() *GoogleBooksProvider {
	return &GoogleBooksProvider{client: &http.Client{}}
}

func (p *GoogleBooksProvider) ID() string { return "google" }

func (p *GoogleBooksProvider) Search(ctx context.Context, q ports.MetadataQuery) ([]ports.MetadataResult, error) {
	query := buildGoogleQuery(q)
	if query == "" {
		return nil, nil
	}

	reqURL := "https://www.googleapis.com/books/v1/volumes?maxResults=10&q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("google books: build request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google books: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google books: HTTP %d", resp.StatusCode)
	}

	var payload googleVolumesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("google books: decode: %w", err)
	}

	results := make([]ports.MetadataResult, 0, len(payload.Items))
	for _, item := range payload.Items {
		results = append(results, googleItemToResult(item))
	}
	return results, nil
}

func buildGoogleQuery(q ports.MetadataQuery) string {
	if q.ISBN != "" {
		return "isbn:" + q.ISBN
	}
	if q.Title != "" && q.Author != "" {
		return "intitle:" + q.Title + "+inauthor:" + q.Author
	}
	if q.Title != "" {
		return "intitle:" + q.Title
	}
	if q.Author != "" {
		return "inauthor:" + q.Author
	}
	return q.Q
}

func googleItemToResult(item googleVolumeItem) ports.MetadataResult {
	info := item.VolumeInfo

	var isbn string
	for _, id := range info.IndustryIdentifiers {
		if id.Type == "ISBN_13" {
			isbn = id.Identifier
			break
		}
		if id.Type == "ISBN_10" && isbn == "" {
			isbn = id.Identifier
		}
	}

	coverURL := info.ImageLinks.Thumbnail
	if coverURL != "" {
		// Google returns http:// thumbnails even over HTTPS; upgrade to avoid mixed-content
		coverURL = strings.Replace(coverURL, "http://", "https://", 1)
	}

	lang := info.Language

	return ports.MetadataResult{
		Title:         info.Title,
		Authors:       info.Authors,
		Description:   info.Description,
		CoverURL:      coverURL,
		Publisher:     info.Publisher,
		PublishedDate: info.PublishedDate,
		ISBN:          isbn,
		Language:      lang,
		Tags:          info.Categories,
		ExternalURL:   info.InfoLink,
		Source: ports.MetadataSource{
			ID:   "google",
			Name: "Google Books",
			Link: "https://books.google.com",
		},
	}
}

// JSON response structs for the Google Books API

type googleVolumesResponse struct {
	Items []googleVolumeItem `json:"items"`
}

type googleVolumeItem struct {
	VolumeInfo googleVolumeInfo `json:"volumeInfo"`
}

type googleVolumeInfo struct {
	Title               string                     `json:"title"`
	Authors             []string                   `json:"authors"`
	Description         string                     `json:"description"`
	Publisher           string                     `json:"publisher"`
	PublishedDate       string                     `json:"publishedDate"`
	Language            string                     `json:"language"`
	Categories          []string                   `json:"categories"`
	InfoLink            string                     `json:"infoLink"`
	IndustryIdentifiers []googleIndustryIdentifier `json:"industryIdentifiers"`
	ImageLinks          googleImageLinks           `json:"imageLinks"`
}

type googleIndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

type googleImageLinks struct {
	Thumbnail string `json:"thumbnail"`
}
