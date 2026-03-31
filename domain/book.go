package domain

import "time"

// FileType distinguishes the kind of resource stored.
type FileType string

const (
	FileTypeEPUB FileType = "epub"
	FileTypePDF  FileType = "pdf"
	FileTypeURL  FileType = "url"
)

// BookMetadata holds bibliographic information extracted from the file or supplied manually.
type BookMetadata struct {
	ISBN        string     `json:"isbn,omitempty"`
	Publisher   string     `json:"publisher,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Language    string     `json:"language,omitempty"`
	Genres      []string   `json:"genres,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
}

// Book is the central domain object representing an item in the library.
type Book struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	Author         string       `json:"author"`
	Description    string       `json:"description"`
	CoverPath      string       `json:"cover_path"`
	FilePath       string       `json:"file_path"`
	FileType       FileType     `json:"file_type"`
	FileSize       int64        `json:"file_size"`
	Metadata       BookMetadata `json:"metadata"`
	ZealotTicketID *string      `json:"zealot_ticket_id,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// Chapter represents a single readable unit within an EPUB.
type Chapter struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Content string `json:"content"` // HTML
}
