// Package epub provides an EPUB metadata parser using only the standard library.
// An EPUB file is a ZIP archive containing an OPF package document with Dublin Core metadata.
package epub

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"alexandria/app/ports"
	"alexandria/domain"
)

type Parser struct{}

// New returns a BookParser for EPUB files.
func New() ports.BookParser {
	return &Parser{}
}

// ParseMetadata opens the EPUB at filePath and extracts bibliographic metadata.
func (p *Parser) ParseMetadata(_ context.Context, filePath string) (*domain.Book, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open epub: %w", err)
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return nil, err
	}

	opf, err := parseOPF(r, opfPath)
	if err != nil {
		return nil, err
	}

	book := &domain.Book{
		FileType: domain.FileTypeEPUB,
	}

	if len(opf.Metadata.Titles) > 0 {
		book.Title = strings.TrimSpace(opf.Metadata.Titles[0])
	}
	if len(opf.Metadata.Creators) > 0 {
		book.Author = strings.TrimSpace(opf.Metadata.Creators[0])
	}
	if len(opf.Metadata.Descriptions) > 0 {
		book.Description = strings.TrimSpace(opf.Metadata.Descriptions[0])
	}

	book.Metadata = domain.BookMetadata{
		Language:  firstOrEmpty(opf.Metadata.Languages),
		Publisher: firstOrEmpty(opf.Metadata.Publishers),
		Genres:    opf.Metadata.Subjects,
	}

	for _, id := range opf.Metadata.Identifiers {
		if strings.EqualFold(id.Scheme, "ISBN") || strings.EqualFold(id.Scheme, "isbn") {
			book.Metadata.ISBN = id.Value
		}
	}

	for _, d := range opf.Metadata.Dates {
		t := parseDate(d)
		if t != nil {
			book.Metadata.PublishedAt = t
			break
		}
	}

	return book, nil
}

// ExtractCover copies the cover image from the EPUB to destPath.
// Returns nil (no error) if no cover is found.
func (p *Parser) ExtractCover(_ context.Context, filePath string, destPath string) error {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("open epub: %w", err)
	}
	defer r.Close()

	opfPath, err := findOPFPath(r)
	if err != nil {
		return nil // non-fatal: no OPF
	}

	opf, err := parseOPF(r, opfPath)
	if err != nil {
		return nil // non-fatal
	}

	coverHref := findCoverHref(opf)
	if coverHref == "" {
		return nil // no cover found — acceptable
	}

	// Resolve href relative to the OPF file's directory
	opfDir := path.Dir(opfPath)
	coverZipPath := path.Join(opfDir, coverHref)

	var coverFile *zip.File
	for _, f := range r.File {
		if f.Name == coverZipPath || f.Name == coverHref {
			coverFile = f
			break
		}
	}
	if coverFile == nil {
		return nil // cover image not found in zip
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("mkdir for cover: %w", err)
	}

	rc, err := coverFile.Open()
	if err != nil {
		return fmt.Errorf("open cover in zip: %w", err)
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create cover file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// --- internal XML structs ---

type opfPackage struct {
	XMLName  xml.Name    `xml:"package"`
	Metadata opfMetadata `xml:"metadata"`
	Manifest opfManifest `xml:"manifest"`
}

type opfMetadata struct {
	Titles       []string        `xml:"http://purl.org/dc/elements/1.1/ title"`
	Creators     []string        `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Descriptions []string        `xml:"http://purl.org/dc/elements/1.1/ description"`
	Publishers   []string        `xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Dates        []string        `xml:"http://purl.org/dc/elements/1.1/ date"`
	Identifiers  []opfIdentifier `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Languages    []string        `xml:"http://purl.org/dc/elements/1.1/ language"`
	Subjects     []string        `xml:"http://purl.org/dc/elements/1.1/ subject"`
	Metas        []opfMeta       `xml:"meta"`
}

type opfIdentifier struct {
	Value  string `xml:",chardata"`
	Scheme string `xml:"scheme,attr"`
}

type opfMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

type opfManifest struct {
	Items []opfItem `xml:"item"`
}

type opfItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

// findOPFPath reads META-INF/container.xml to locate the package document.
func findOPFPath(r *zip.ReadCloser) (string, error) {
	type rootFile struct {
		FullPath string `xml:"full-path,attr"`
	}
	type container struct {
		RootFiles []rootFile `xml:"rootfiles>rootfile"`
	}

	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open container.xml: %w", err)
			}
			defer rc.Close()

			var c container
			if err := xml.NewDecoder(rc).Decode(&c); err != nil {
				return "", fmt.Errorf("decode container.xml: %w", err)
			}
			if len(c.RootFiles) == 0 {
				return "", fmt.Errorf("no rootfile in container.xml")
			}
			return c.RootFiles[0].FullPath, nil
		}
	}
	return "", fmt.Errorf("META-INF/container.xml not found in epub")
}

func parseOPF(r *zip.ReadCloser, opfPath string) (*opfPackage, error) {
	for _, f := range r.File {
		if f.Name == opfPath {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open opf: %w", err)
			}
			defer rc.Close()

			var pkg opfPackage
			if err := xml.NewDecoder(rc).Decode(&pkg); err != nil {
				return nil, fmt.Errorf("decode opf: %w", err)
			}
			return &pkg, nil
		}
	}
	return nil, fmt.Errorf("OPF file %q not found in epub", opfPath)
}

func findCoverHref(opf *opfPackage) string {
	// Prefer item with properties="cover-image" (EPUB 3)
	for _, item := range opf.Manifest.Items {
		if item.Properties == "cover-image" {
			return item.Href
		}
	}
	// Fall back to item with id containing "cover" and an image media type
	for _, item := range opf.Manifest.Items {
		if strings.Contains(strings.ToLower(item.ID), "cover") &&
			strings.HasPrefix(item.MediaType, "image/") {
			return item.Href
		}
	}
	// Check meta name="cover" (EPUB 2)
	for _, meta := range opf.Metadata.Metas {
		if meta.Name == "cover" {
			for _, item := range opf.Manifest.Items {
				if item.ID == meta.Content {
					return item.Href
				}
			}
		}
	}
	return ""
}

func firstOrEmpty(ss []string) string {
	if len(ss) > 0 {
		return strings.TrimSpace(ss[0])
	}
	return ""
}

func parseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	formats := []string{"2006-01-02", "2006-01", "2006", time.RFC3339}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}
