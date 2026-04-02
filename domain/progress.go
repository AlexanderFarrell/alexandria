package domain

import "time"

// ReadingProgress tracks where a user left off in a book.
type ReadingProgress struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	BookID            string     `json:"book_id"`
	SectionID         string     `json:"section_id"`
	SectionProgress   float64    `json:"section_progress"` // 0.0–1.0 within the current section
	BlockIndex        *int       `json:"block_index,omitempty"`
	Percentage        float64    `json:"percentage"` // 0.0–1.0
	Rating            *int       `json:"rating,omitempty"` // 1–5; nil means unrated
	ZealotProgressRef *string    `json:"zealot_progress_ref,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	LastReadAt        time.Time  `json:"last_read_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}
