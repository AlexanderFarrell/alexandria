package sqlite

import "time"

// BookModel is the GORM persistence model for a Book.
// Metadata fields are flattened into the table; genres and tags are stored as JSON arrays.
type BookModel struct {
	ID              string `gorm:"primaryKey"`
	Title           string `gorm:"not null;index"`
	Author          string `gorm:"index"`
	Description     string
	CoverPath       string
	FilePath        string `gorm:"not null"`
	FileType        string `gorm:"not null"`
	FileSize        int64
	MetaISBN        string
	MetaPublisher   string
	MetaPublishedAt *time.Time
	MetaLanguage    string
	MetaGenres      string // JSON: ["fiction","adventure"]
	MetaTags        string // JSON: ["classic","recommended"]
	ZealotTicketID  *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (BookModel) TableName() string { return "books" }

// UserModel is the GORM persistence model for a User.
type UserModel struct {
	ID           string  `gorm:"primaryKey"`
	Username     string  `gorm:"uniqueIndex;not null"`
	Email        *string `gorm:"uniqueIndex"` // pointer so absent emails are NULL, not "" (avoids UNIQUE conflict)
	PasswordHash string  `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserModel) TableName() string { return "users" }

// ProgressModel is the GORM persistence model for ReadingProgress.
// The combination (UserID, BookID) is unique — one progress record per user per book.
type ProgressModel struct {
	ID                string `gorm:"primaryKey"`
	UserID            string `gorm:"not null;uniqueIndex:idx_user_book"`
	BookID            string `gorm:"not null;uniqueIndex:idx_user_book"`
	CFI               string // legacy epub.js field kept for compatibility with old rows
	SectionID         string
	SectionProgress   float64
	BlockIndex        *int
	Percentage        float64
	Rating            *int // 1–5; nil means unrated
	ZealotProgressRef *string
	StartedAt         time.Time
	LastReadAt        time.Time
	FinishedAt        *time.Time
}

func (ProgressModel) TableName() string { return "reading_progress" }

// BookListModel is the GORM persistence model for a BookList.
type BookListModel struct {
	ID          string `gorm:"primaryKey"`
	UserID      string `gorm:"not null;index"`
	Name        string `gorm:"not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (BookListModel) TableName() string { return "book_lists" }

// BookListItemModel links a book to a list.
// The combination (ListID, BookID) is unique.
type BookListItemModel struct {
	ListID  string `gorm:"not null;uniqueIndex:idx_list_book"`
	BookID  string `gorm:"not null;uniqueIndex:idx_list_book"`
	AddedAt time.Time
}

func (BookListItemModel) TableName() string { return "book_list_items" }

// BookLinkModel is the GORM persistence model for a BookLink.
type BookLinkModel struct {
	ID        string `gorm:"primaryKey"`
	BookID    string `gorm:"not null;index"`
	Label     string `gorm:"not null"`
	URL       string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (BookLinkModel) TableName() string { return "book_links" }
