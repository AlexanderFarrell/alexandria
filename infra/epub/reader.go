package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"alexandria/domain"
)

const (
	epubNavType               = "toc"
	epubNavDocRole            = "doc-toc"
	epubMediaTypeNCX          = "application/x-dtbncx+xml"
	epubMediaTypeCSS          = "text/css"
	epubReaderSectionLinkAttr = "data-reader-section-id"
	epubReaderFragmentAttr    = "data-reader-fragment"
)

var (
	cssURLPattern    = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)
	cssImportPattern = regexp.MustCompile(`@import\s+(?:url\(\s*)?['"]?([^'")\s]+)['"]?\s*\)?`)
)

type readerPackage struct {
	Manifest readerManifest `xml:"manifest"`
	Spine    readerSpine    `xml:"spine"`
}

type readerManifest struct {
	Items []readerManifestItem `xml:"item"`
}

type readerManifestItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

type readerSpine struct {
	TOC      string           `xml:"toc,attr"`
	ItemRefs []readerSpineRef `xml:"itemref"`
}

type readerSpineRef struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

type rawNavItem struct {
	Label    string
	Href     string
	Children []rawNavItem
}

type ReaderBook struct {
	zipReader      *zip.ReadCloser
	opfPath        string
	opfDir         string
	files          map[string]*zip.File
	manifestByID   map[string]readerManifestItem
	manifestByPath map[string]readerManifestItem
	sections       []readerSection
	sectionByID    map[string]readerSection
	sectionByPath  map[string]readerSection
	nav            []domain.ReaderNavItem
	firstSectionID string
}

type readerSection struct {
	ID        string
	Title     string
	Href      string
	FullPath  string
	MediaType string
	Index     int
	Linear    bool
}

func OpenReaderBook(filePath string) (*ReaderBook, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open epub: %w", err)
	}

	book, err := newReaderBook(reader)
	if err != nil {
		reader.Close()
		return nil, err
	}

	return book, nil
}

func newReaderBook(reader *zip.ReadCloser) (*ReaderBook, error) {
	opfPath, err := findOPFPath(reader)
	if err != nil {
		return nil, err
	}

	opf, err := parseReaderOPF(reader, opfPath)
	if err != nil {
		return nil, err
	}

	book := &ReaderBook{
		zipReader:      reader,
		opfPath:        opfPath,
		opfDir:         path.Dir(opfPath),
		files:          make(map[string]*zip.File, len(reader.File)),
		manifestByID:   make(map[string]readerManifestItem, len(opf.Manifest.Items)),
		manifestByPath: make(map[string]readerManifestItem, len(opf.Manifest.Items)),
		sectionByID:    make(map[string]readerSection),
		sectionByPath:  make(map[string]readerSection),
	}

	for _, file := range reader.File {
		book.files[normalizeZipPath(file.Name)] = file
	}

	for _, item := range opf.Manifest.Items {
		fullPath := resolveRelativeZipPath(book.opfPath, item.Href)
		entry := readerManifestItem{
			ID:         item.ID,
			Href:       item.Href,
			MediaType:  item.MediaType,
			Properties: item.Properties,
		}
		book.manifestByID[item.ID] = entry
		book.manifestByPath[fullPath] = entry
	}

	for index, itemRef := range opf.Spine.ItemRefs {
		item, ok := book.manifestByID[itemRef.IDRef]
		if !ok {
			continue
		}
		fullPath := resolveRelativeZipPath(book.opfPath, item.Href)
		section := readerSection{
			ID:        item.ID,
			Href:      item.Href,
			FullPath:  fullPath,
			MediaType: item.MediaType,
			Index:     index,
			Linear:    !strings.EqualFold(strings.TrimSpace(itemRef.Linear), "no"),
		}
		book.sections = append(book.sections, section)
	}

	if len(book.sections) == 0 {
		return nil, fmt.Errorf("epub has no readable spine sections")
	}

	for _, section := range book.sections {
		book.sectionByID[section.ID] = section
		book.sectionByPath[section.FullPath] = section
	}

	labelHints := make(map[string]string, len(book.sections))
	rawNav, navBasePath := book.parseRawNav(opf)
	if len(rawNav) > 0 {
		book.collectNavLabels(rawNav, navBasePath, labelHints)
	}

	for index := range book.sections {
		section := book.sections[index]
		section.Title = labelHints[section.ID]
		if section.Title == "" {
			raw, err := book.readFile(section.FullPath)
			if err == nil {
				section.Title = firstHeadingTitle(raw)
			}
		}
		if section.Title == "" {
			section.Title = titleFromPath(section.Href)
		}
		book.sections[index] = section
		book.sectionByID[section.ID] = section
		book.sectionByPath[section.FullPath] = section
	}

	seenSections := make(map[string]bool, len(book.sections))
	if len(rawNav) > 0 {
		book.nav = book.normalizeNav(rawNav, navBasePath, seenSections)
	}
	for _, section := range book.sections {
		if !section.Linear || seenSections[section.ID] {
			continue
		}
		book.nav = append(book.nav, domain.ReaderNavItem{
			Label:     section.Title,
			SectionID: section.ID,
		})
		seenSections[section.ID] = true
	}

	book.firstSectionID = book.sections[0].ID
	return book, nil
}

func (b *ReaderBook) Close() error {
	if b == nil || b.zipReader == nil {
		return nil
	}
	return b.zipReader.Close()
}

func (b *ReaderBook) Manifest() *domain.ReaderManifest {
	sections := make([]domain.ReaderSectionSummary, 0, len(b.sections))
	for _, section := range b.sections {
		sections = append(sections, domain.ReaderSectionSummary{
			ID:    section.ID,
			Title: section.Title,
			Index: section.Index,
		})
	}

	nav := make([]domain.ReaderNavItem, len(b.nav))
	copy(nav, b.nav)

	return &domain.ReaderManifest{
		Sections:       sections,
		Nav:            nav,
		FirstSectionID: b.firstSectionID,
	}
}

func (b *ReaderBook) Section(sectionID string, assetURL func(string) string) (*domain.ReaderSection, error) {
	section, ok := b.sectionByID[sectionID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	raw, err := b.readFile(section.FullPath)
	if err != nil {
		return nil, err
	}

	html, err := b.rewriteSectionHTML(section, raw, assetURL)
	if err != nil {
		return nil, fmt.Errorf("rewrite section %q: %w", section.ID, err)
	}

	var prevSectionID string
	var nextSectionID string
	if section.Index > 0 {
		prevSectionID = b.sections[section.Index-1].ID
	}
	if section.Index+1 < len(b.sections) {
		nextSectionID = b.sections[section.Index+1].ID
	}

	return &domain.ReaderSection{
		ID:            section.ID,
		Title:         section.Title,
		SectionIndex:  section.Index,
		PrevSectionID: prevSectionID,
		NextSectionID: nextSectionID,
		HTML:          html,
	}, nil
}

func (b *ReaderBook) Asset(assetPath string, assetURL func(string) string) ([]byte, string, error) {
	assetPath = normalizeZipPath(assetPath)
	if assetPath == "" {
		return nil, "", domain.ErrNotFound
	}
	if _, ok := b.manifestByPath[assetPath]; !ok {
		return nil, "", domain.ErrNotFound
	}
	if _, isSection := b.sectionByPath[assetPath]; isSection {
		return nil, "", domain.ErrNotFound
	}

	raw, err := b.readFile(assetPath)
	if err != nil {
		return nil, "", err
	}

	contentType := b.assetContentType(assetPath, raw)
	if strings.HasPrefix(contentType, epubMediaTypeCSS) || strings.HasSuffix(strings.ToLower(assetPath), ".css") {
		raw = []byte(rewriteCSSReferences(assetPath, string(raw), assetURL))
		contentType = epubMediaTypeCSS + "; charset=utf-8"
	}

	return raw, contentType, nil
}

func (b *ReaderBook) parseRawNav(opf *readerPackage) ([]rawNavItem, string) {
	for _, item := range opf.Manifest.Items {
		if !hasProperty(item.Properties, "nav") {
			continue
		}

		fullPath := resolveRelativeZipPath(b.opfPath, item.Href)
		raw, err := b.readFile(fullPath)
		if err != nil {
			continue
		}
		items, err := parseXHTMLNav(raw)
		if err == nil && len(items) > 0 {
			return items, fullPath
		}
	}

	ncxPath := ""
	if opf.Spine.TOC != "" {
		if item, ok := b.manifestByID[opf.Spine.TOC]; ok {
			ncxPath = resolveRelativeZipPath(b.opfPath, item.Href)
		}
	}
	if ncxPath == "" {
		for _, item := range opf.Manifest.Items {
			if strings.EqualFold(item.MediaType, epubMediaTypeNCX) {
				ncxPath = resolveRelativeZipPath(b.opfPath, item.Href)
				break
			}
		}
	}
	if ncxPath == "" {
		return nil, ""
	}

	raw, err := b.readFile(ncxPath)
	if err != nil {
		return nil, ""
	}
	items, err := parseNCXNav(raw)
	if err != nil {
		return nil, ""
	}
	return items, ncxPath
}

func (b *ReaderBook) collectNavLabels(items []rawNavItem, basePath string, labels map[string]string) {
	for _, item := range items {
		label := normalizeText(item.Label)
		if label != "" && item.Href != "" {
			targetPath, _, remote := resolveLinkTarget(basePath, item.Href)
			if !remote {
				if section, ok := b.sectionByPath[targetPath]; ok && labels[section.ID] == "" {
					labels[section.ID] = label
				}
			}
		}
		if len(item.Children) > 0 {
			b.collectNavLabels(item.Children, basePath, labels)
		}
	}
}

func (b *ReaderBook) normalizeNav(items []rawNavItem, basePath string, seenSections map[string]bool) []domain.ReaderNavItem {
	seenTargets := make(map[string]bool)
	return b.normalizeNavItems(items, basePath, seenSections, seenTargets)
}

func (b *ReaderBook) normalizeNavItems(
	items []rawNavItem,
	basePath string,
	seenSections map[string]bool,
	seenTargets map[string]bool,
) []domain.ReaderNavItem {
	result := make([]domain.ReaderNavItem, 0, len(items))
	for _, item := range items {
		normalized := domain.ReaderNavItem{
			Label: normalizeText(item.Label),
		}

		if item.Href != "" {
			targetPath, fragment, remote := resolveLinkTarget(basePath, item.Href)
			if !remote {
				if section, ok := b.sectionByPath[targetPath]; ok {
					targetKey := section.ID + "#" + fragment
					if !seenTargets[targetKey] {
						normalized.SectionID = section.ID
						normalized.Fragment = fragment
						seenTargets[targetKey] = true
						seenSections[section.ID] = true
					}
				}
			}
		}

		if len(item.Children) > 0 {
			normalized.Children = b.normalizeNavItems(item.Children, basePath, seenSections, seenTargets)
		}

		if normalized.Label == "" {
			if normalized.SectionID != "" {
				normalized.Label = b.sectionByID[normalized.SectionID].Title
			} else if len(normalized.Children) > 0 {
				normalized.Label = normalized.Children[0].Label
			}
		}

		if normalized.Label == "" {
			continue
		}
		if normalized.SectionID == "" && len(normalized.Children) == 0 {
			continue
		}

		result = append(result, normalized)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return b.navSortKey(result[i]) < b.navSortKey(result[j])
	})
	return result
}

func (b *ReaderBook) navSortKey(item domain.ReaderNavItem) string {
	if item.SectionID == "" {
		if len(item.Children) == 0 {
			return fmt.Sprintf("%08d", len(b.sections))
		}
		return b.navSortKey(item.Children[0])
	}

	section := b.sectionByID[item.SectionID]
	return fmt.Sprintf("%08d:%s:%s", section.Index, section.ID, item.Fragment)
}

func (b *ReaderBook) rewriteSectionHTML(section readerSection, raw []byte, assetURL func(string) string) (string, error) {
	decoder := newXMLDecoder(bytes.NewReader(raw))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)

	skipDepth := 0
	inStyle := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if skipDepth > 0 {
			switch token.(type) {
			case xml.StartElement:
				skipDepth++
			case xml.EndElement:
				skipDepth--
			}
			continue
		}

		switch current := token.(type) {
		case xml.StartElement:
			if shouldStripElement(current.Name.Local) {
				skipDepth = 1
				continue
			}

			if strings.EqualFold(current.Name.Local, "style") {
				inStyle = true
			}

			rewritten := xml.StartElement{Name: current.Name, Attr: make([]xml.Attr, 0, len(current.Attr)+2)}
			for _, attr := range current.Attr {
				if shouldStripAttribute(attr.Name.Local) {
					continue
				}

				if strings.EqualFold(current.Name.Local, "a") && strings.EqualFold(attr.Name.Local, "href") {
					href, targetSectionID, fragment, keep := b.rewriteAnchorReference(section, attr.Value, assetURL)
					if !keep {
						continue
					}

					rewritten.Attr = append(rewritten.Attr, xml.Attr{
						Name:  xml.Name{Local: "href"},
						Value: href,
					})
					if targetSectionID != "" {
						rewritten.Attr = append(rewritten.Attr, xml.Attr{
							Name:  xml.Name{Local: epubReaderSectionLinkAttr},
							Value: targetSectionID,
						})
						if fragment != "" {
							rewritten.Attr = append(rewritten.Attr, xml.Attr{
								Name:  xml.Name{Local: epubReaderFragmentAttr},
								Value: fragment,
							})
						}
					}
					continue
				}

				value, keep := b.rewriteAttribute(section, current.Name.Local, attr.Name.Local, attr.Value, assetURL)
				if !keep {
					continue
				}

				attr.Value = value
				rewritten.Attr = append(rewritten.Attr, attr)
			}

			if err := encoder.EncodeToken(rewritten); err != nil {
				return "", err
			}
		case xml.EndElement:
			if strings.EqualFold(current.Name.Local, "style") {
				inStyle = false
			}
			if err := encoder.EncodeToken(current); err != nil {
				return "", err
			}
		case xml.CharData:
			data := current
			if inStyle {
				data = xml.CharData([]byte(rewriteCSSReferences(section.FullPath, string(current), assetURL)))
			}
			if err := encoder.EncodeToken(data); err != nil {
				return "", err
			}
		default:
			if err := encoder.EncodeToken(token); err != nil {
				return "", err
			}
		}
	}

	if err := encoder.Flush(); err != nil {
		return "", err
	}
	return output.String(), nil
}

func (b *ReaderBook) rewriteAnchorReference(
	section readerSection,
	rawValue string,
	assetURL func(string) string,
) (string, string, string, bool) {
	if isDataURL(rawValue) {
		return rawValue, "", "", true
	}

	targetPath, fragment, remote := resolveLinkTarget(section.FullPath, rawValue)
	if remote {
		return "", "", "", false
	}
	if fragment != "" && targetPath == section.FullPath {
		return "#" + fragment, "", "", true
	}
	if targetPath == section.FullPath {
		return "#", "", "", true
	}
	if targetSection, ok := b.sectionByPath[targetPath]; ok {
		return "#", targetSection.ID, fragment, true
	}
	if targetPath != "" {
		return assetURL(targetPath), "", "", true
	}
	return "", "", "", false
}

func (b *ReaderBook) rewriteAttribute(
	section readerSection,
	elementName string,
	attrName string,
	rawValue string,
	assetURL func(string) string,
) (string, bool) {
	if rawValue == "" {
		return rawValue, true
	}

	if strings.EqualFold(attrName, "style") {
		return rewriteCSSReferences(section.FullPath, rawValue, assetURL), true
	}
	if strings.EqualFold(attrName, "srcset") {
		return rewriteSrcSet(section.FullPath, rawValue, assetURL), true
	}

	if strings.EqualFold(attrName, epubReaderSectionLinkAttr) ||
		strings.EqualFold(attrName, epubReaderFragmentAttr) {
		return "", false
	}

	if isAssetAttribute(elementName, attrName) {
		if isDataURL(rawValue) {
			return rawValue, true
		}
		targetPath, _, remote := resolveLinkTarget(section.FullPath, rawValue)
		if remote || targetPath == "" {
			return "", false
		}
		if _, ok := b.sectionByPath[targetPath]; ok {
			return "", false
		}
		return assetURL(targetPath), true
	}

	return rawValue, true
}

func (b *ReaderBook) readFile(filePath string) ([]byte, error) {
	filePath = normalizeZipPath(filePath)
	file, ok := b.files[filePath]
	if !ok {
		return nil, domain.ErrNotFound
	}

	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open epub file %q: %w", filePath, err)
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read epub file %q: %w", filePath, err)
	}
	return raw, nil
}

func (b *ReaderBook) assetContentType(assetPath string, raw []byte) string {
	if item, ok := b.manifestByPath[assetPath]; ok && item.MediaType != "" {
		return item.MediaType
	}
	contentType := http.DetectContentType(raw)
	if contentType == "text/plain; charset=utf-8" && strings.HasSuffix(strings.ToLower(assetPath), ".css") {
		return epubMediaTypeCSS + "; charset=utf-8"
	}
	return contentType
}

func parseReaderOPF(reader *zip.ReadCloser, opfPath string) (*readerPackage, error) {
	for _, file := range reader.File {
		if normalizeZipPath(file.Name) != normalizeZipPath(opfPath) {
			continue
		}
		content, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open opf: %w", err)
		}
		defer content.Close()

		var pkg readerPackage
		decoder := newXMLDecoder(content)
		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decode opf: %w", err)
		}
		return &pkg, nil
	}
	return nil, fmt.Errorf("opf file %q not found in epub", opfPath)
}

func parseXHTMLNav(raw []byte) ([]rawNavItem, error) {
	decoder := newXMLDecoder(bytes.NewReader(raw))

	type navBuilder struct {
		item     *rawNavItem
		labelBuf strings.Builder
	}

	var root []*navBuilder
	stack := make([]*navBuilder, 0)
	inTOCNav := false
	navDepth := 0
	captureLabel := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch current := token.(type) {
		case xml.StartElement:
			if strings.EqualFold(current.Name.Local, "nav") {
				if !inTOCNav && elementHasTOCType(current.Attr) {
					inTOCNav = true
					navDepth = 1
					continue
				}
				if inTOCNav {
					navDepth++
				}
			}
			if !inTOCNav {
				continue
			}
			switch strings.ToLower(current.Name.Local) {
			case "li":
				builder := &navBuilder{item: &rawNavItem{}}
				if len(stack) == 0 {
					root = append(root, builder)
				} else {
					parent := stack[len(stack)-1]
					parent.item.Children = append(parent.item.Children, *builder.item)
					childIndex := len(parent.item.Children) - 1
					builder.item = &parent.item.Children[childIndex]
				}
				stack = append(stack, builder)
			case "a", "span":
				if len(stack) == 0 {
					continue
				}
				captureLabel = true
				if strings.EqualFold(current.Name.Local, "a") {
					if href, ok := attrValue(current.Attr, "href"); ok {
						stack[len(stack)-1].item.Href = href
					}
				}
			}
		case xml.EndElement:
			if strings.EqualFold(current.Name.Local, "nav") && inTOCNav {
				navDepth--
				if navDepth == 0 {
					inTOCNav = false
				}
			}
			if !inTOCNav && navDepth == 0 {
				continue
			}
			switch strings.ToLower(current.Name.Local) {
			case "a", "span":
				captureLabel = false
				if len(stack) > 0 {
					stack[len(stack)-1].item.Label = normalizeText(stack[len(stack)-1].labelBuf.String())
				}
			case "li":
				if len(stack) == 0 {
					continue
				}
				builder := stack[len(stack)-1]
				builder.item.Label = normalizeText(builder.labelBuf.String())
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if captureLabel && len(stack) > 0 {
				stack[len(stack)-1].labelBuf.Write([]byte(current))
			}
		}
	}

	items := make([]rawNavItem, 0, len(root))
	for _, item := range root {
		if item != nil && item.item != nil {
			items = append(items, *item.item)
		}
	}
	return items, nil
}

func parseNCXNav(raw []byte) ([]rawNavItem, error) {
	var doc struct {
		NavMap struct {
			Points []ncxNavPoint `xml:"navPoint"`
		} `xml:"navMap"`
	}
	decoder := newXMLDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}

	items := make([]rawNavItem, 0, len(doc.NavMap.Points))
	for _, point := range doc.NavMap.Points {
		items = append(items, point.toRaw())
	}
	return items, nil
}

type ncxNavPoint struct {
	NavLabel struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Points []ncxNavPoint `xml:"navPoint"`
}

func (p ncxNavPoint) toRaw() rawNavItem {
	children := make([]rawNavItem, 0, len(p.Points))
	for _, child := range p.Points {
		children = append(children, child.toRaw())
	}
	return rawNavItem{
		Label:    normalizeText(p.NavLabel.Text),
		Href:     p.Content.Src,
		Children: children,
	}
}

func resolveLinkTarget(basePath string, raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", true
	}
	if parsed.Scheme != "" && parsed.Scheme != "file" {
		return "", "", true
	}
	if parsed.Host != "" {
		return "", "", true
	}

	fragment := parsed.Fragment
	if parsed.Path == "" {
		return normalizeZipPath(basePath), fragment, false
	}

	return resolveRelativeZipPath(basePath, parsed.Path), fragment, false
}

func resolveRelativeZipPath(basePath string, href string) string {
	baseDir := path.Dir(normalizeZipPath(basePath))
	if strings.HasPrefix(strings.TrimSpace(href), "/") {
		return normalizeZipPath(strings.TrimPrefix(strings.TrimSpace(href), "/"))
	}
	return normalizeZipPath(path.Join(baseDir, href))
}

func normalizeZipPath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || value == "." {
		return ""
	}
	cleaned := path.Clean(value)
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func titleFromPath(filePath string) string {
	filePath = normalizeZipPath(filePath)
	base := path.Base(filePath)
	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = normalizeText(base)
	if base == "" {
		return "Untitled Section"
	}
	return base
}

func firstHeadingTitle(raw []byte) string {
	decoder := newXMLDecoder(bytes.NewReader(raw))
	headingDepth := 0
	var label strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}

		switch current := token.(type) {
		case xml.StartElement:
			if headingDepth == 0 && isHeadingElement(current.Name.Local) {
				headingDepth = 1
				continue
			}
			if headingDepth > 0 {
				headingDepth++
			}
		case xml.EndElement:
			if headingDepth > 0 {
				headingDepth--
				if headingDepth == 0 {
					return normalizeText(label.String())
				}
			}
		case xml.CharData:
			if headingDepth > 0 {
				label.Write([]byte(current))
			}
		}
	}
	return ""
}

func newXMLDecoder(reader io.Reader) *xml.Decoder {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity
	return decoder
}

func hasProperty(properties string, want string) bool {
	for _, part := range strings.Fields(strings.ToLower(properties)) {
		if part == strings.ToLower(want) {
			return true
		}
	}
	return false
}

func elementHasTOCType(attrs []xml.Attr) bool {
	for _, attr := range attrs {
		value := strings.ToLower(strings.TrimSpace(attr.Value))
		switch strings.ToLower(attr.Name.Local) {
		case "type":
			if strings.Contains(value, epubNavType) {
				return true
			}
		case "role":
			if strings.Contains(value, epubNavDocRole) {
				return true
			}
		}
	}
	return false
}

func attrValue(attrs []xml.Attr, local string) (string, bool) {
	for _, attr := range attrs {
		if strings.EqualFold(attr.Name.Local, local) {
			return attr.Value, true
		}
	}
	return "", false
}

func shouldStripElement(name string) bool {
	switch strings.ToLower(name) {
	case "script", "base":
		return true
	default:
		return false
	}
}

func shouldStripAttribute(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "on")
}

func isAssetAttribute(elementName string, attrName string) bool {
	switch strings.ToLower(attrName) {
	case "src", "poster", "data", "xlink:href":
		return true
	case "href":
		switch strings.ToLower(elementName) {
		case "link", "image", "img", "use":
			return true
		}
	}
	return false
}

func isHeadingElement(name string) bool {
	switch strings.ToLower(name) {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func rewriteSrcSet(basePath string, value string, assetURL func(string) string) string {
	parts := strings.Split(value, ",")
	rewritten := make([]string, 0, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		fields := strings.Fields(candidate)
		if len(fields) == 0 {
			continue
		}
		if isDataURL(fields[0]) {
			rewritten = append(rewritten, candidate)
			continue
		}
		targetPath, _, remote := resolveLinkTarget(basePath, fields[0])
		if remote || targetPath == "" {
			continue
		}
		fields[0] = assetURL(targetPath)
		rewritten = append(rewritten, strings.Join(fields, " "))
	}
	return strings.Join(rewritten, ", ")
}

func rewriteCSSReferences(basePath string, css string, assetURL func(string) string) string {
	css = cssURLPattern.ReplaceAllStringFunc(css, func(match string) string {
		submatches := cssURLPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		target := strings.TrimSpace(submatches[1])
		if isDataURL(target) {
			return match
		}
		targetPath, fragment, remote := resolveLinkTarget(basePath, target)
		if remote {
			return "none"
		}
		if targetPath == "" && fragment != "" {
			return match
		}
		if targetPath == "" {
			return match
		}
		return fmt.Sprintf("url(%q)", assetURL(targetPath))
	})

	css = cssImportPattern.ReplaceAllStringFunc(css, func(match string) string {
		submatches := cssImportPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		if isDataURL(submatches[1]) {
			return match
		}
		targetPath, _, remote := resolveLinkTarget(basePath, submatches[1])
		if remote || targetPath == "" {
			return ""
		}
		return fmt.Sprintf("@import url(%q)", assetURL(targetPath))
	})

	return css
}

func isDataURL(value string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "data:")
}
