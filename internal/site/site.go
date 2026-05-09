package site

import (
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"html"
	"html/template"
	"regexp"

	"osg/internal/date"
	"osg/internal/frontmatter"
	"osg/internal/markdown"
	"osg/internal/publish"
)

// Translation describes a single alternate-language version of a page.
type Translation struct {
	Lang      string
	Path      string
	Permalink string
	Title     string
}

type Page struct {
	Title        string
	MenuTitle    string
	Slug         string
	Path         string
	Permalink    string
	SourcePath   string
	Date         time.Time
	Updated      time.Time
	PublishAt    time.Time // Future-dated publish time; zero means publish immediately.
	Draft        bool
	Menu         bool
	Author       string
	Image        string
	ImageCredit  string
	Summary      string
	Content      string
	RawContent   string
	Template     string
	Lang         string
	WordCount    int
	ReadingTime  int
	Series       string
	SeriesOrder  int
	Robots       string
	NoIndex      bool
	CanonicalURL string
	Keywords     []string
	Taxonomies   map[string][]string
	Extra        map[string]any
	Translations []Translation
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

// LinkTranslations cross-references pages that share the same slug across
// different languages.  Two pages are considered translations of each other
// when they have the same Slug but different Lang values.
func (s *Site) LinkTranslations() {
	// Build a slug -> []*Page index.
	bySlug := make(map[string][]*Page)
	for _, p := range s.Pages {
		if p.Slug == "" {
			continue
		}
		bySlug[p.Slug] = append(bySlug[p.Slug], p)
	}

	for _, group := range bySlug {
		if len(group) < 2 {
			continue
		}

		// Check that the group actually has pages in more than one language.
		langs := make(map[string]bool)
		for _, p := range group {
			langs[p.Lang] = true
		}
		if len(langs) < 2 {
			continue
		}

		// Link each page to all other-language pages in the group.
		for _, page := range group {
			for _, other := range group {
				if other.Lang == page.Lang {
					continue
				}
				page.Translations = append(page.Translations, Translation{
					Lang:      other.Lang,
					Path:      other.Path,
					Permalink: other.Permalink,
					Title:     other.Title,
				})
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

	// Extract osg block early so osg.title can participate in title resolution.
	osg := publish.GetOSGBlock(fm)

	// Title precedence: osg.title > fm.title > fm.name > slug
	title := pickString(osg, "title")
	if title == "" {
		title = pickString(fm, "title", "name")
	}
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

	// Resolve image_credit: osg.image_credit > top-level image_credit
	imageCredit := ""
	if osg != nil {
		imageCredit = pickString(osg, "image_credit")
	}
	if imageCredit == "" {
		imageCredit = pickString(fm, "image_credit")
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

	// Resolve summary: osg.summary / osg.abstract > top-level
	// summary/description/excerpt. Both osg.summary and osg.abstract
	// are accepted because the field name was historically "abstract"
	// but "summary" reads more naturally and the UI exposes that
	// label in its editor.
	pageSummary := ""
	if osg != nil {
		pageSummary = pickString(osg, "summary", "abstract")
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

	// Resolve SEO directives: osg.* > top-level
	pageRobots := ""
	if osg != nil {
		pageRobots = pickString(osg, "robots")
	}
	if pageRobots == "" {
		pageRobots = pickString(fm, "robots")
	}

	pageNoIndex := false
	if osg != nil {
		pageNoIndex = pickBool(osg, "noindex")
	}
	if !pageNoIndex {
		pageNoIndex = pickBool(fm, "noindex")
	}

	pageCanonical := ""
	if osg != nil {
		pageCanonical = pickString(osg, "canonical_url", "canonical")
	}
	if pageCanonical == "" {
		pageCanonical = pickString(fm, "canonical_url", "canonical")
	}

	var pageKeywords []string
	if osg != nil {
		pageKeywords = toStringSlice(osg["keywords"])
	}
	if len(pageKeywords) == 0 {
		pageKeywords = toStringSlice(fm["keywords"])
	}

	pagePublishAt := pickTime(osg, "publish_at")
	if pagePublishAt.IsZero() {
		pagePublishAt = pickTime(fm, "publish_at")
	}

	page := &Page{
		Title:        title,
		MenuTitle:    pickString(fm, "menu_title"),
		Slug:         slug,
		Path:         pagePath,
		Permalink:    permalink,
		SourcePath:   filePath,
		Date:         fileDate,
		PublishAt:    pagePublishAt,
		Draft:        pickBool(fm, "draft"),
		Menu:         pickBool(fm, "menu"),
		Author:       pageAuthor,
		Image:        pageImage,
		ImageCredit:  imageCredit,
		Summary:      pageSummary,
		Content:      contentHTML,
		RawContent:   string(body),
		Series:       pickString(fm, "series"),
		SeriesOrder:  pickInt(fm, "series_order"),
		Template:     pickString(fm, "template"),
		Lang:         pickString(fm, "lang", "language"),
		Robots:       pageRobots,
		NoIndex:      pageNoIndex,
		CanonicalURL: pageCanonical,
		Keywords:     pageKeywords,
		Taxonomies:   pickTaxonomies(fm),
		Extra:        fm,
	}

	// Store featured in Extra so Section.View() can find it
	if isFeatured {
		if page.Extra == nil {
			page.Extra = map[string]any{}
		}
		page.Extra["featured"] = true
	}

	page.WordCount = len(strings.Fields(string(body)))
	page.ReadingTime = max(page.WordCount/200, 1)

	return page, nil, nil
}

// IsScheduled returns true when PublishAt is set and still in the future
// at the moment of the call. Schedulers and build pipelines use this to
// hide a page until its release date passes.
func (p *Page) IsScheduled() bool {
	return !p.PublishAt.IsZero() && p.PublishAt.After(time.Now())
}

func (p *Page) View() map[string]any {
	var translations []map[string]any
	for _, t := range p.Translations {
		translations = append(translations, map[string]any{
			"lang":      t.Lang,
			"path":      t.Path,
			"permalink": t.Permalink,
			"title":     t.Title,
		})
	}

	return map[string]any{
		"title":             p.Title,
		"menu_title":        p.MenuTitle,
		"slug":              p.Slug,
		"path":              p.Path,
		"permalink":         p.Permalink,
		"date":              p.Date,
		"updated":           p.Updated,
		"publish_at":        p.PublishAt,
		"draft":             p.Draft,
		"menu":              p.Menu,
		"author":            p.Author,
		"image":             p.Image,
		"image_credit":      p.ImageCredit,
		"image_credit_html": template.HTML(renderCreditHTML(p.ImageCredit)),
		"summary":           p.Summary,
		"content":           template.HTML(p.Content),
		"raw_content":       p.RawContent,
		"word_count":        p.WordCount,
		"reading_time":      p.ReadingTime,
		"series":            p.Series,
		"series_order":      p.SeriesOrder,
		"robots":            p.Robots,
		"noindex":           p.NoIndex,
		"canonical_url":     p.CanonicalURL,
		"keywords":          p.Keywords,
		"taxonomies":        p.Taxonomies,
		"extra":             p.Extra,
		"lang":              p.Lang,
		"translations":      translations,
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
	featuredIdx := -1
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
		"is_root":       s.IsRoot,
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
	if rest, ok := strings.CutSuffix(rel, ".md"); ok {
		return "/" + rest + "/"
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

// renderCreditHTML converts a markdown-link-style credit string into safe HTML.
// It turns [text](url) into <a> tags with rel="noopener noreferrer" and escapes
// everything else.
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

func renderCreditHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var buf strings.Builder
	last := 0
	for _, m := range mdLinkRe.FindAllStringSubmatchIndex(raw, -1) {
		// m[0]:m[1] = full match, m[2]:m[3] = text, m[4]:m[5] = url
		buf.WriteString(html.EscapeString(raw[last:m[0]]))
		text := html.EscapeString(raw[m[2]:m[3]])
		href := html.EscapeString(raw[m[4]:m[5]])
		buf.WriteString(`<a href="` + href + `" rel="noopener noreferrer">` + text + `</a>`)
		last = m[1]
	}
	buf.WriteString(html.EscapeString(raw[last:]))
	return buf.String()
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

// pickTime parses a frontmatter value as a date/time using the same
// layout list as date.Derive. Returns the zero time when not found or
// unparseable.
func pickTime(fm map[string]any, key string) time.Time {
	if fm == nil {
		return time.Time{}
	}
	val, ok := fm[key]
	if !ok {
		return time.Time{}
	}
	if t, ok := date.Parse(val); ok {
		return t
	}
	return time.Time{}
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

func pickInt(fm map[string]any, key string) int {
	if fm == nil {
		return 0
	}
	val, ok := fm[key]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
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
			maps.Copy(out, v)
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
	merged := make([]string, 0, len(existing)+len(values))
	merged = append(merged, existing...)
	merged = append(merged, values...)
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
	value = strings.TrimPrefix(value, "#")
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
