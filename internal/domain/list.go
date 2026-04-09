package domain

import "time"

// BookList is a user-curated collection of books.
type BookList struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BookListItem links a book to a list.
type BookListItem struct {
	ListID  string    `json:"list_id"`
	BookID  string    `json:"book_id"`
	AddedAt time.Time `json:"added_at"`
}
