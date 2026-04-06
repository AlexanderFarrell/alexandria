package domain

// ReaderManifest describes the EPUB structure exposed to the reader client.
type ReaderManifest struct {
	Sections       []ReaderSectionSummary `json:"sections"`
	Nav            []ReaderNavItem        `json:"nav"`
	FirstSectionID string                 `json:"first_section_id"`
}

// ReaderSectionSummary is a lightweight section entry used by the reader shell.
type ReaderSectionSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Index int    `json:"index"`
}

// ReaderNavItem describes a navigation target exposed to the reader client.
type ReaderNavItem struct {
	Label     string          `json:"label"`
	SectionID string          `json:"section_id,omitempty"`
	Fragment  string          `json:"fragment,omitempty"`
	Children  []ReaderNavItem `json:"children,omitempty"`
}

// ReaderSection is the rendered section payload returned to the client.
type ReaderSection struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	SectionIndex  int    `json:"section_index"`
	PrevSectionID string `json:"prev_section_id,omitempty"`
	NextSectionID string `json:"next_section_id,omitempty"`
	HTML          string `json:"html"`
}
