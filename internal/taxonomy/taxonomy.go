package taxonomy

import (
	"path"
	"sort"
	"strings"

	"osg/internal/config"
	"osg/internal/site"
	"osg/internal/slug"
)

type Index struct {
	Config config.TaxonomyConfig
	Terms  map[string]*Term
}

type Term struct {
	Name      string
	Slug      string
	Path      string
	Permalink string
	Pages     []*site.Page
}

func Build(cfgs []config.TaxonomyConfig, pages []*site.Page, baseURL string) map[string]*Index {
	indices := map[string]*Index{}
	for _, cfg := range cfgs {
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			continue
		}
		indices[name] = &Index{Config: cfg, Terms: map[string]*Term{}}
	}

	if len(indices) == 0 {
		return indices
	}

	// Pre-compute excluded terms per taxonomy (lowercased for
	// case-insensitive matching).
	excluded := map[string]map[string]bool{}
	for name, index := range indices {
		if len(index.Config.ExcludeTerms) > 0 {
			set := make(map[string]bool, len(index.Config.ExcludeTerms))
			for _, t := range index.Config.ExcludeTerms {
				set[strings.ToLower(strings.TrimSpace(t))] = true
			}
			excluded[name] = set
		}
	}

	for _, page := range pages {
		for kind, terms := range page.Taxonomies {
			index, ok := indices[kind]
			if !ok {
				continue
			}
			for _, termName := range terms {
				termName = strings.TrimSpace(termName)
				if termName == "" {
					continue
				}
				if ex, ok := excluded[kind]; ok && ex[strings.ToLower(termName)] {
					continue
				}

				termSlug := slug.Slugify(termName)
				if termSlug == "" {
					termSlug = "term"
				}

				term, exists := index.Terms[termSlug]
				if !exists {
					termPath := ensureTrailingSlash(path.Join("/", kind, termSlug))
					term = &Term{
						Name:      termName,
						Slug:      termSlug,
						Path:      termPath,
						Permalink: buildPermalink(baseURL, termPath),
						Pages:     []*site.Page{},
					}
					index.Terms[termSlug] = term
				}

				term.Pages = append(term.Pages, page)
			}
		}
	}

	for _, index := range indices {
		for _, term := range index.Terms {
			sortPages(term.Pages)
		}
	}

	return indices
}

func (i *Index) TermsSorted() []*Term {
	if i == nil {
		return nil
	}

	terms := make([]*Term, 0, len(i.Terms))
	for _, term := range i.Terms {
		terms = append(terms, term)
	}

	sort.SliceStable(terms, func(a, b int) bool {
		return strings.ToLower(terms[a].Name) < strings.ToLower(terms[b].Name)
	})

	return terms
}

func ConfigView(cfg config.TaxonomyConfig) map[string]any {
	return map[string]any{
		"name":          cfg.Name,
		"paginate_by":   cfg.PaginateBy,
		"paginate_path": cfg.PaginatePath,
		"feed":          cfg.Feed,
		"render":        cfg.Render,
		"exclude_terms": cfg.ExcludeTerms,
	}
}

func TermView(term *Term) map[string]any {
	pages := make([]map[string]any, 0, len(term.Pages))
	for _, page := range term.Pages {
		pages = append(pages, page.View())
	}

	return map[string]any{
		"name":       term.Name,
		"slug":       term.Slug,
		"path":       term.Path,
		"permalink":  term.Permalink,
		"pages":      pages,
		"page_count": len(term.Pages),
	}
}

func TermViews(terms []*Term) []map[string]any {
	out := make([]map[string]any, 0, len(terms))
	for _, term := range terms {
		out = append(out, TermView(term))
	}
	return out
}

// FilterPageTaxonomies removes excluded terms from every page's Taxonomies
// map so that templates (e.g. "Publicado en:", card pills) never show them.
// Call this right after Build().
func FilterPageTaxonomies(cfgs []config.TaxonomyConfig, pages []*site.Page) {
	// Build the same excluded-terms sets used in Build().
	excluded := map[string]map[string]bool{}
	for _, cfg := range cfgs {
		if len(cfg.ExcludeTerms) == 0 {
			continue
		}
		set := make(map[string]bool, len(cfg.ExcludeTerms))
		for _, t := range cfg.ExcludeTerms {
			set[strings.ToLower(strings.TrimSpace(t))] = true
		}
		excluded[cfg.Name] = set
	}
	if len(excluded) == 0 {
		return
	}

	for _, page := range pages {
		for kind, terms := range page.Taxonomies {
			ex, ok := excluded[kind]
			if !ok {
				continue
			}
			filtered := make([]string, 0, len(terms))
			for _, t := range terms {
				if !ex[strings.ToLower(strings.TrimSpace(t))] {
					filtered = append(filtered, t)
				}
			}
			if len(filtered) == 0 {
				delete(page.Taxonomies, kind)
			} else {
				page.Taxonomies[kind] = filtered
			}
		}
	}
}

func sortPages(pages []*site.Page) {
	sort.SliceStable(pages, func(i, j int) bool {
		left := pages[i]
		right := pages[j]
		if !left.Date.Equal(right.Date) {
			return left.Date.After(right.Date)
		}
		return strings.ToLower(left.Title) < strings.ToLower(right.Title)
	})
}

func buildPermalink(baseURL string, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return path
	}
	return strings.TrimRight(baseURL, "/") + path
}
