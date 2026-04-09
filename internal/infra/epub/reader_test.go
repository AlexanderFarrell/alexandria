package epub

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"alexandria/domain"
)

func TestReaderBookBuildsSpineOrderedManifestFromNavDoc(t *testing.T) {
	filePath := writeTempEPUB(t, buildReaderFixtureEPUB(t, true))

	readerBook, err := OpenReaderBook(filePath)
	if err != nil {
		t.Fatalf("open reader book: %v", err)
	}
	defer readerBook.Close()

	manifest := readerBook.Manifest()
	if manifest.FirstSectionID != "chap-1" {
		t.Fatalf("expected first section chap-1, got %q", manifest.FirstSectionID)
	}
	if len(manifest.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(manifest.Sections))
	}
	if manifest.Sections[0].Title != "First Chapter" {
		t.Fatalf("expected first section title from nav label, got %q", manifest.Sections[0].Title)
	}
	if manifest.Sections[2].Title != "Third Chapter" {
		t.Fatalf("expected missing nav title to fall back to section heading, got %q", manifest.Sections[2].Title)
	}

	if len(manifest.Nav) != 3 {
		t.Fatalf("expected 3 root nav items, got %d", len(manifest.Nav))
	}
	rootIDs := []string{
		manifest.Nav[0].SectionID,
		manifest.Nav[1].SectionID,
		manifest.Nav[2].SectionID,
	}
	wantRootIDs := []string{"chap-1", "chap-2", "chap-3"}
	for index := range wantRootIDs {
		if rootIDs[index] != wantRootIDs[index] {
			t.Fatalf("expected nav root %d to be %q, got %q", index, wantRootIDs[index], rootIDs[index])
		}
	}

	if len(manifest.Nav[0].Children) != 1 {
		t.Fatalf("expected duplicate chapter child entries to be deduplicated, got %d children", len(manifest.Nav[0].Children))
	}
	if manifest.Nav[0].Children[0].Fragment != "note-one" {
		t.Fatalf("expected nested child fragment note-one, got %q", manifest.Nav[0].Children[0].Fragment)
	}
}

func TestReaderBookFallsBackToNCX(t *testing.T) {
	filePath := writeTempEPUB(t, buildReaderFixtureEPUB(t, false))

	readerBook, err := OpenReaderBook(filePath)
	if err != nil {
		t.Fatalf("open reader book: %v", err)
	}
	defer readerBook.Close()

	manifest := readerBook.Manifest()
	if len(manifest.Nav) != 3 {
		t.Fatalf("expected 3 nav items after ncx fallback plus missing spine append, got %d", len(manifest.Nav))
	}
	if manifest.Nav[0].Label != "NCX One" || manifest.Nav[0].SectionID != "chap-1" {
		t.Fatalf("unexpected first ncx nav item: %+v", manifest.Nav[0])
	}
	if manifest.Nav[1].SectionID != "chap-2" {
		t.Fatalf("unexpected second ncx nav item: %+v", manifest.Nav[1])
	}
	if manifest.Nav[2].SectionID != "chap-3" {
		t.Fatalf("expected missing spine section to be appended, got %+v", manifest.Nav[2])
	}
}

func TestReaderBookRewritesSectionAndAssets(t *testing.T) {
	filePath := writeTempEPUB(t, buildReaderFixtureEPUB(t, true))

	readerBook, err := OpenReaderBook(filePath)
	if err != nil {
		t.Fatalf("open reader book: %v", err)
	}
	defer readerBook.Close()

	resolver := func(assetPath string) string {
		return "asset://" + assetPath + "?rt=test-token"
	}

	section, err := readerBook.Section("chap-1", resolver)
	if err != nil {
		t.Fatalf("load section: %v", err)
	}

	if !strings.Contains(section.HTML, `href="asset://OEBPS/styles/book.css?rt=test-token"`) {
		t.Fatalf("expected section html to rewrite stylesheet href, got %s", section.HTML)
	}
	if !strings.Contains(section.HTML, `data-reader-section-id="chap-2"`) {
		t.Fatalf("expected cross-section link rewrite, got %s", section.HTML)
	}
	if !strings.Contains(section.HTML, `data-reader-fragment="part-two"`) {
		t.Fatalf("expected cross-section fragment rewrite, got %s", section.HTML)
	}
	if !strings.Contains(section.HTML, `href="#note-one"`) {
		t.Fatalf("expected same-section anchor to remain local, got %s", section.HTML)
	}
	if strings.Contains(section.HTML, "https://example.com") {
		t.Fatalf("expected remote links to be stripped, got %s", section.HTML)
	}
	if strings.Contains(section.HTML, "<script") {
		t.Fatalf("expected scripts to be stripped, got %s", section.HTML)
	}
	if strings.Contains(section.HTML, "onclick=") {
		t.Fatalf("expected inline handlers to be stripped, got %s", section.HTML)
	}

	cssPayload, contentType, err := readerBook.Asset("OEBPS/styles/book.css", resolver)
	if err != nil {
		t.Fatalf("load css asset: %v", err)
	}
	if !strings.HasPrefix(contentType, "text/css") {
		t.Fatalf("expected css content type, got %q", contentType)
	}
	if !strings.Contains(string(cssPayload), `asset://OEBPS/images/pic.png?rt=test-token`) {
		t.Fatalf("expected css asset URLs to be rewritten, got %s", cssPayload)
	}

	_, imageContentType, err := readerBook.Asset("OEBPS/images/pic.png", resolver)
	if err != nil {
		t.Fatalf("load png asset: %v", err)
	}
	if imageContentType != "image/png" {
		t.Fatalf("expected image/png content type, got %q", imageContentType)
	}

	_, _, err = readerBook.Asset("META-INF/container.xml", resolver)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected arbitrary zip entries to be rejected, got %v", err)
	}
}

func writeTempEPUB(t *testing.T, payload []byte) string {
	t.Helper()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "book.epub")
	if err := os.WriteFile(filePath, payload, 0o644); err != nil {
		t.Fatalf("write temp epub: %v", err)
	}
	return filePath
}

func buildReaderFixtureEPUB(t *testing.T, useNavDoc bool) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	writeZipString(t, zipWriter, "mimetype", "application/epub+zip")
	writeZipString(t, zipWriter, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	manifestNav := `<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>`
	if useNavDoc {
		manifestNav = `<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>`
	}

	writeZipString(t, zipWriter, "OEBPS/content.opf", `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="BookId">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Reader Fixture</dc:title>
    <dc:creator>Fixture Author</dc:creator>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    `+manifestNav+`
    <item id="css" href="styles/book.css" media-type="text/css"/>
    <item id="img" href="images/pic.png" media-type="image/png"/>
    <item id="chap-1" href="text/ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chap-2" href="text/ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="chap-3" href="text/ch3.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="chap-1"/>
    <itemref idref="chap-2"/>
    <itemref idref="chap-3"/>
  </spine>
</package>`)

	writeZipString(t, zipWriter, "OEBPS/nav.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
  <head><title>Nav</title></head>
  <body>
    <nav epub:type="toc">
      <ol>
        <li><a href="text/ch2.xhtml">Second Chapter</a></li>
        <li>
          <a href="text/ch1.xhtml">First Chapter</a>
          <ol>
            <li><a href="text/ch1.xhtml#note-one">First Note</a></li>
            <li><a href="text/ch1.xhtml#note-one">Duplicate Note</a></li>
          </ol>
        </li>
      </ol>
    </nav>
  </body>
</html>`)

	writeZipString(t, zipWriter, "OEBPS/toc.ncx", `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="navPoint-1" playOrder="1">
      <navLabel><text>NCX One</text></navLabel>
      <content src="text/ch1.xhtml"/>
    </navPoint>
    <navPoint id="navPoint-2" playOrder="2">
      <navLabel><text>NCX Two</text></navLabel>
      <content src="text/ch2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	writeZipString(t, zipWriter, "OEBPS/styles/book.css", `body { background-image: url('../images/pic.png'); }
.remote { background-image: url('https://example.com/blocked.png'); }`)

	writeZipBytes(t, zipWriter, "OEBPS/images/pic.png", []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	})

	writeZipString(t, zipWriter, "OEBPS/text/ch1.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Chapter One</title>
    <link rel="stylesheet" href="../styles/book.css"/>
  </head>
  <body>
    <h1>Chapter One</h1>
    <p onclick="evil()">Alpha paragraph.</p>
    <p><img src="../images/pic.png" alt="pic"/></p>
    <p><a href="ch2.xhtml#part-two">Cross chapter</a></p>
    <p><a href="#note-one">Local note</a></p>
    <p><a href="https://example.com">Blocked remote</a></p>
    <p id="note-one">Footnote target.</p>
    <script>alert('blocked')</script>
  </body>
</html>`)

	writeZipString(t, zipWriter, "OEBPS/text/ch2.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter Two</title></head>
  <body>
    <h1 id="part-two">Second Chapter</h1>
    <p>Beta paragraph.</p>
  </body>
</html>`)

	writeZipString(t, zipWriter, "OEBPS/text/ch3.xhtml", `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Third</title></head>
  <body>
    <h1>Third Chapter</h1>
    <p>Gamma paragraph.</p>
  </body>
</html>`)

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}

	return buffer.Bytes()
}

func writeZipString(t *testing.T, writer *zip.Writer, name string, payload string) {
	t.Helper()
	writeZipBytes(t, writer, name, []byte(payload))
}

func writeZipBytes(t *testing.T, writer *zip.Writer, name string, payload []byte) {
	t.Helper()
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %s: %v", name, err)
	}
	if _, err := entry.Write(payload); err != nil {
		t.Fatalf("write zip entry %s: %v", name, err)
	}
}
