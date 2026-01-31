package site

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"html/template"

	"osg/internal/date"
	"osg/internal/frontmatter"
	"osg/internal/markdown"
)

type Page struct {
	Title      string
	Slug       string
	Path       string
	Permalink  string
	Date       time.Time
	Updated    time.Time
	Draft      bool
	Summary    string
	Content    string
	RawContent string
	Template   string
	Lang       string
	Taxonomies map[string][]string
	Extra      map[string]any
}

type Section struct {
	Title       string
	Slug        string
	Path        string
	Permalink   string
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
		sectionPath := parentPath(page.Path)
		section := s.ensureSection(sectionPath)
		section.Pages = append(section.Pages, page)
	}

	for path, section := range s.Sections {
		if section.IsRoot {
			continue
		}
		parentPath := parentPath(path)
		parent := s.ensureSection(parentPath)
		parent.Subsections = append(parent.Subsections, section)
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

	section := &Section{
		Title:  "",
		Slug:   strings.Trim(path.Base(sectionPath), "/"),
		Path:   sectionPath,
		Pages:  []*Page{},
		IsRoot: sectionPath == "/",
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

	if isSection {
		section := &Section{
			Title:     title,
			Slug:      slug,
			Path:      pagePath,
			Permalink: permalink,
			Content:   contentHTML,
			Template:  pickString(fm, "template"),
			Extra:     fm,
			IsRoot:    pagePath == "/",
		}
		return nil, section, nil
	}

	page := &Page{
		Title:      title,
		Slug:       slug,
		Path:       pagePath,
		Permalink:  permalink,
		Date:       fileDate,
		Draft:      pickBool(fm, "draft"),
		Summary:    pickString(fm, "summary", "description", "excerpt"),
		Content:    contentHTML,
		RawContent: string(body),
		Template:   pickString(fm, "template"),
		Lang:       pickString(fm, "lang", "language"),
		Taxonomies: pickTaxonomies(fm),
		Extra:      fm,
	}

	return page, nil, nil
}

func (p *Page) View() map[string]any {
	return map[string]any{
		"title":       p.Title,
		"slug":        p.Slug,
		"path":        p.Path,
		"permalink":   p.Permalink,
		"date":        p.Date,
		"updated":     p.Updated,
		"draft":       p.Draft,
		"summary":     p.Summary,
		"content":     template.HTML(p.Content),
		"raw_content": p.RawContent,
		"taxonomies":  p.Taxonomies,
		"extra":       p.Extra,
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

	return map[string]any{
		"title":       s.Title,
		"slug":        s.Slug,
		"path":        s.Path,
		"permalink":   s.Permalink,
		"content":     template.HTML(s.Content),
		"pages":       pages,
		"subsections": subsections,
		"extra":       s.Extra,
	}
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
	if !ok || value == nil {
		return out
	}

	switch v := value.(type) {
	case map[string]any:
		for key, raw := range v {
			out[key] = toStringSlice(raw)
		}
	case map[string][]string:
		return v
	}

	return out
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
