package ports

import "context"

// MetadataResult is a normalized book record from one external provider.
type MetadataResult struct {
	Title         string
	Authors       []string
	Description   string
	CoverURL      string
	Publisher     string
	PublishedDate string // raw: "YYYY", "YYYY-MM", or "YYYY-MM-DD"
	ISBN          string
	Language      string
	Tags          []string
	Source        MetadataSource
	ExternalURL   string
}

// MetadataSource identifies which provider produced a result.
type MetadataSource struct {
	ID   string
	Name string
	Link string
}

// MetadataQuery carries the search parameters sent by the client.
type MetadataQuery struct {
	Q      string // free-text fallback
	Title  string
	Author string
	ISBN   string
}

// MetadataProvider fetches book metadata from one external source.
// Implementations must respect the context deadline for timeouts.
type MetadataProvider interface {
	ID() string
	Search(ctx context.Context, query MetadataQuery) ([]MetadataResult, error)
}
