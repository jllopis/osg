package site

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"html/template"

	"osg/internal/date"
	"osg/internal/frontmatter"
	"osg/internal/markdown"
	"osg/internal/publish"
)

type Page struct {
	Title       string
	Slug        string
	Path        string
	Permalink   string
	SourcePath  string
	Date        time.Time
	Updated     time.Time
	Draft       bool
	Menu        bool
	Author      string
	Image       string
	Summary     string
	Content     string
	RawContent  string
	Template    string
	Lang        string
	WordCount   int
	ReadingTime int
	Taxonomies  map[string][]string
	Extra       map[string]any
}

type Section struct {
	Title       string
	Slug        string
	Path        string
	Permalink   string
	SourcePath  string
	Content     string
	Template    string
	Pages       []*Page
	Subsections []*Section
	Extra       map[string]any
	IsRoot      bool
}

type Site struct {
	Pages    []*Page
	Sections map[string]*Section
	Root     *Section
}

func New() *Site {
	root := &Section{
		Title:  "Home",
		Slug:   "",
		Path:   "/",
		IsRoot: true,
		Pages:  []*Page{},
	}

	return &Site{
		Pages:    []*Page{},
		Sections: map[string]*Section{root.Path: root},
		Root:     root,
	}
}

func (s *Site) AddPage(page *Page) {
	s.Pages = append(s.Pages, page)
}

// MenuPages returns all pages marked with Menu: true, suitable for
// navigation menu rendering in templates.
func (s *Site) MenuPages() []*Page {
	var out []*Page
	for _, page := range s.Pages {
		if page.Menu {
			out = append(out, page)
		}
	}
	return out
}

func (s *Site) AddSection(section *Section) {
	if section == nil {
		return
	}

	if existing, ok := s.Sections[section.Path]; ok {
		if section.Content != "" {
			existing.Content = section.Content
		}
		if section.Title != "" {
			existing.Title = section.Title
		}
		if section.Template != "" {
			existing.Template = section.Template
		}
		if len(section.Extra) > 0 {
			existing.Extra = section.Extra
		}
		return
	}

	s.Sections[section.Path] = section
}

func (s *Site) BuildHierarchy() {
	for _, page := range s.Pages {
		if page.Menu {
			continue // menu pages are standalone; don't add them to any section listing
		}
		sectionPath := parentPath(page.Path)
		section := s.ensureSection(sectionPath)
		section.Pages = append(section.Pages, page)
	}

	// Link subsections to their parents. Because intermediate sections
	// (e.g. /2023/09/) may be created on the fly by ensureSection, we
	// repeat until no new sections appear (a section is "linked" once
	// it has been attached to its parent).
	linked := make(map[string]bool)
	linked["/"] = true // root has no parent to link to
	for {
		progress := false
		for spath, section := range s.Sections {
			if linked[spath] {
				continue
			}
			pp := parentPath(spath)
			parent := s.ensureSection(pp)
			parent.Subsections = append(parent.Subsections, section)
			linked[spath] = true
			progress = true
		}
		if !progress {
			break
		}
	}

	// Sort pages in every section by date descending (most recent first).
	for _, section := range s.Sections {
		sort.Slice(section.Pages, func(i, j int) bool {
			return section.Pages[i].Date.After(section.Pages[j].Date)
		})
	}

	// Sort site-level pages by date descending as well.
	sort.Slice(s.Pages, func(i, j int) bool {
		return s.Pages[i].Date.After(s.Pages[j].Date)
	})

	// If the root section has no direct pages (common with date-based
	// content_layout like "{date}/{slug}"), populate it with all site pages
	// so that the index.html template can list recent posts.
	// Menu pages are excluded — they are standalone navigation items.
	if root, ok := s.Sections["/"]; ok && len(root.Pages) == 0 && len(s.Pages) > 0 {
		for _, p := range s.Pages {
			if !p.Menu {
				root.Pages = append(root.Pages, p)
			}
		}
	}
}

func (s *Site) View() map[string]any {
	pages := make([]map[string]any, 0, len(s.Pages))
	for _, page := range s.Pages {
		pages = append(pages, page.View())
	}

	var root map[string]any
	if s.Root != nil {
		root = s.Root.View()
	}

	return map[string]any{
		"pages": pages,
		"root":  root,
	}
}

func (s *Site) ensureSection(sectionPath string) *Section {
	if existing, ok := s.Sections[sectionPath]; ok {
		return existing
	}

	slug := strings.Trim(path.Base(sectionPath), "/")
	section := &Section{
		Title:     slug,
		Slug:      slug,
		Path:      sectionPath,
		Permalink: sectionPath,
		Pages:     []*Page{},
		IsRoot:    sectionPath == "/",
	}

	s.Sections[sectionPath] = section
	return section
}

func ParseFile(contentDir string, baseURL string, filePath string) (*Page, *Section, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	fm, body, _, err := frontmatter.SplitFrontmatter(data)
	if err != nil {
		return nil, nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	rel, err := filepath.Rel(contentDir, filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("rel path: %w", err)
	}

	rel = filepath.ToSlash(rel)
	base := path.Base(rel)
	isSection := base == "_index.md"

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("stat file: %w", err)
	}

	fileDate := date.Derive(fm, info)
	pagePath := pathFromRel(rel)
	permalink := buildPermalink(baseURL, pagePath)
	contentHTML, err := markdown.Render(body)
	if err != nil {
		return nil, nil, fmt.Errorf("render markdown: %w", err)
	}

	title := pickString(fm, "title", "name")
	slug := pickString(fm, "slug")
	if slug == "" {
		if isSection {
			if rel == "_index.md" {
				slug = ""
			} else {
				slug = strings.Trim(path.Base(path.Dir(rel)), "/")
			}
		} else {
			slug = strings.TrimSuffix(base, ".md")
		}
	}

	if title == "" {
		title = strings.Trim(slug, "-")
	}

	// Extract osg block for osg.image and osg.featured overrides
	osg := publish.GetOSGBlock(fm)

	if isSection {
		section := &Section{
			Title:      title,
			Slug:       slug,
			Path:       pagePath,
			Permalink:  permalink,
			SourcePath: filePath,
			Content:    contentHTML,
			Template:   pickString(fm, "template"),
			Extra:      fm,
			IsRoot:     pagePath == "/",
		}
		return nil, section, nil
	}

	// Resolve image: osg.image > top-level image/cover/banner > frontmatter "featured"
	pageImage := ""
	if osg != nil {
		pageImage = pickString(osg, "image")
	}
	if pageImage == "" {
		pageImage = pickString(fm, "image", "cover", "banner")
	}

	// Resolve featured flag: osg.featured > top-level featured > extra.featured
	isFeatured := false
	if osg != nil {
		isFeatured = pickBool(osg, "featured")
	}
	if !isFeatured {
		isFeatured = pickBool(fm, "featured")
	}
	if !isFeatured {
		if extra, ok := fm["extra"].(map[string]any); ok {
			if v, ok := extra["featured"]; ok {
				if b, ok := v.(bool); ok {
					isFeatured = b
				}
			}
		}
	}

	// Resolve summary: osg.abstract > top-level summary/description/excerpt
	pageSummary := ""
	if osg != nil {
		pageSummary = pickString(osg, "abstract")
	}
	if pageSummary == "" {
		pageSummary = pickString(fm, "summary", "description", "excerpt")
	}

	// Resolve author: osg.author > top-level author
	pageAuthor := ""
	if osg != nil {
		pageAuthor = pickString(osg, "author")
	}
	if pageAuthor == "" {
		pageAuthor = pickString(fm, "author")
	}

	page := &Page{
		Title:      title,
		Slug:       slug,
		Path:       pagePath,
		Permalink:  permalink,
		SourcePath: filePath,
		Date:       fileDate,
		Draft:      pickBool(fm, "draft"),
		Menu:       pickBool(fm, "menu"),
		Author:     pageAuthor,
		Image:      pageImage,
		Summary:    pageSummary,
		Content:    contentHTML,
		RawContent: string(body),
		Template:   pickString(fm, "template"),
		Lang:       pickString(fm, "lang", "language"),
		Taxonomies: pickTaxonomies(fm),
		Extra:      fm,
	}

	// Store featured in Extra so Section.View() can find it
	if isFeatured {
		if page.Extra == nil {
			page.Extra = map[string]any{}
		}
		page.Extra["featured"] = true
	}

	page.WordCount = len(strings.Fields(string(body)))
	page.ReadingTime = page.WordCount / 200
	if page.ReadingTime < 1 {
		page.ReadingTime = 1
	}

	return page, nil, nil
}

func (p *Page) View() map[string]any {
	return map[string]any{
		"title":        p.Title,
		"slug":         p.Slug,
		"path":         p.Path,
		"permalink":    p.Permalink,
		"date":         p.Date,
		"updated":      p.Updated,
		"draft":        p.Draft,
		"menu":         p.Menu,
		"author":       p.Author,
		"image":        p.Image,
		"summary":      p.Summary,
		"content":      template.HTML(p.Content),
		"raw_content":  p.RawContent,
		"word_count":   p.WordCount,
		"reading_time": p.ReadingTime,
		"taxonomies":   p.Taxonomies,
		"extra":        p.Extra,
		"lang":         p.Lang,
	}
}

func (s *Section) View() map[string]any {
	pages := make([]map[string]any, 0, len(s.Pages))
	for _, page := range s.Pages {
		pages = append(pages, page.View())
	}

	subsections := make([]map[string]any, 0, len(s.Subsections))
	for _, section := range s.Subsections {
		subsections = append(subsections, section.View())
	}

	// Determine the featured page: the most-recent page with
	// extra.featured == true becomes the hero; any other featured pages
	// are promoted to the top of the list (before non-featured), keeping
	// their relative date order.  If no page is explicitly featured the
	// most-recent page is used as hero.
	var featured map[string]any
	var featuredIdx int = -1
	for i, page := range s.Pages {
		if val, ok := page.Extra["featured"]; ok {
			if b, ok := val.(bool); ok && b {
				featured = pages[i]
				featuredIdx = i
				break
			}
		}
	}
	if featured == nil && len(s.Pages) > 0 {
		featured = pages[0]
		featuredIdx = 0
	}

	// Reorder the page list: featured posts first (excluding the hero),
	// then non-featured — both groups keep their original date order.
	if featuredIdx >= 0 {
		reordered := make([]map[string]any, 0, len(pages))
		var rest []map[string]any
		for i, pv := range pages {
			if i == featuredIdx {
				continue // skip the hero
			}
			if isFeaturedPage(s.Pages[i]) {
				reordered = append(reordered, pv)
			} else {
				rest = append(rest, pv)
			}
		}
		reordered = append(reordered, rest...)
		pages = reordered
	}

	return map[string]any{
		"title":         s.Title,
		"slug":          s.Slug,
		"path":          s.Path,
		"permalink":     s.Permalink,
		"content":       template.HTML(s.Content),
		"pages":         pages,
		"subsections":   subsections,
		"extra":         s.Extra,
		"featured_page": featured,
		"has_source":    s.SourcePath != "",
	}
}

func isFeaturedPage(p *Page) bool {
	if val, ok := p.Extra["featured"]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func pathFromRel(rel string) string {
	if rel == "index.md" || rel == "_index.md" {
		return "/"
	}
	if strings.HasSuffix(rel, "/index.md") {
		return "/" + strings.TrimSuffix(rel, "index.md")
	}
	if strings.HasSuffix(rel, "/_index.md") {
		return "/" + strings.TrimSuffix(rel, "_index.md")
	}
	if strings.HasSuffix(rel, ".md") {
		return "/" + strings.TrimSuffix(rel, ".md") + "/"
	}
	if strings.HasPrefix(rel, "/") {
		return rel
	}
	return "/" + rel
}

func parentPath(p string) string {
	clean := strings.TrimSuffix(p, "/")
	if clean == "" {
		return "/"
	}
	parent := path.Dir(clean)
	if parent == "." {
		parent = "/"
	}
	if !strings.HasSuffix(parent, "/") {
		parent += "/"
	}
	return parent
}

func buildPermalink(baseURL string, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return path
	}
	return strings.TrimRight(baseURL, "/") + path
}

func pickString(fm map[string]any, keys ...string) string {
	for _, key := range keys {
		if fm == nil {
			return ""
		}
		if val, ok := fm[key]; ok {
			if str, ok := val.(string); ok {
				return strings.TrimSpace(str)
			}
		}
	}
	return ""
}

func pickBool(fm map[string]any, key string) bool {
	if fm == nil {
		return false
	}
	val, ok := fm[key]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func pickTaxonomies(fm map[string]any) map[string][]string {
	out := map[string][]string{}
	if fm == nil {
		return out
	}

	value, ok := fm["taxonomies"]
	if ok && value != nil {
		switch v := value.(type) {
		case map[string]any:
			for key, raw := range v {
				out[key] = toStringSlice(raw)
			}
		case map[string][]string:
			for key, raw := range v {
				out[key] = raw
			}
		}
	}

	mergeTaxonomy(out, "tags", toStringSlice(fm["tags"]))
	mergeTaxonomy(out, "area", toStringSlice(fm["area"]))
	mergeTaxonomy(out, "type", toStringSlice(fm["type"]))

	return out
}

func mergeTaxonomy(out map[string][]string, key string, values []string) {
	if len(values) == 0 {
		return
	}
	values = normalizeTerms(values)
	existing := out[key]
	merged := append(existing, values...)
	out[key] = uniqueStrings(merged)
}

func normalizeTerms(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if term := normalizeTerm(value); term != "" {
			out = append(out, term)
		}
	}
	return out
}

func normalizeTerm(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "#") {
		value = strings.TrimPrefix(value, "#")
	}
	if strings.HasPrefix(value, "[[") && strings.HasSuffix(value, "]]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(value, "[["), "]]")
		inner = strings.TrimSpace(inner)
		if inner == "" {
			return ""
		}
		if parts := strings.SplitN(inner, "|", 2); len(parts) > 0 {
			value = strings.TrimSpace(parts[0])
		} else {
			value = inner
		}
	}
	return strings.TrimSpace(value)
}

func toStringSlice(value any) []string {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return compactStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	case string:
		s := strings.TrimSpace(v)
		if s != "" {
			return []string{s}
		}
	}

	return nil
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, s := range values {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
