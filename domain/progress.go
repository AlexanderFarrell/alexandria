package domain

import "time"

// ReadingProgress tracks where a user left off in a book.
type ReadingProgress struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	BookID            string     `json:"book_id"`
	CFI               string     `json:"cfi"`        // epub.js Canonical Fragment Identifier
	Percentage        float64    `json:"percentage"` // 0.0–1.0
	Rating            *int       `json:"rating,omitempty"` // 1–5; nil means unrated
	ZealotProgressRef *string    `json:"zealot_progress_ref,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	LastReadAt        time.Time  `json:"last_read_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}
