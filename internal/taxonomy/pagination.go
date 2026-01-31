package taxonomy

import (
	"fmt"
	"math"
	"path"
	"strings"

	"osg/internal/site"
)

type Paginator struct {
	PaginateBy   int
	BaseURL      string
	NumberPagers int
	First        string
	Last         string
	Previous     string
	Next         string
	Pages        []*site.Page
	CurrentIndex int
	TotalPages   int
}

func BuildPaginator(pages []*site.Page, paginateBy int, baseURL string, paginatePath string) []Paginator {
	if paginateBy <= 0 || len(pages) <= paginateBy {
		return nil
	}

	if paginatePath == "" {
		paginatePath = "page"
	}

	baseURL = ensureTrailingSlash(baseURL)
	count := int(math.Ceil(float64(len(pages)) / float64(paginateBy)))
	out := make([]Paginator, 0, count)

	for i := 0; i < count; i++ {
		start := i * paginateBy
		end := start + paginateBy
		if end > len(pages) {
			end = len(pages)
		}

		p := Paginator{
			PaginateBy:   paginateBy,
			BaseURL:      baseURL,
			NumberPagers: count,
			Pages:        pages[start:end],
			CurrentIndex: i + 1,
			TotalPages:   count,
			First:        baseURL,
			Last:         ensureTrailingSlash(path.Join(baseURL, paginatePath, fmt.Sprintf("%d", count))),
		}

		if i > 0 {
			if i == 1 {
				p.Previous = baseURL
			} else {
				p.Previous = ensureTrailingSlash(path.Join(baseURL, paginatePath, fmt.Sprintf("%d", i)))
			}
		}

		if i < count-1 {
			p.Next = ensureTrailingSlash(path.Join(baseURL, paginatePath, fmt.Sprintf("%d", i+2)))
		}

		out = append(out, p)
	}

	return out
}

func PaginatorView(p Paginator) map[string]any {
	pages := make([]map[string]any, 0, len(p.Pages))
	for _, page := range p.Pages {
		pages = append(pages, page.View())
	}

	return map[string]any{
		"paginate_by":   p.PaginateBy,
		"base_url":      p.BaseURL,
		"number_pagers": p.NumberPagers,
		"first":         p.First,
		"last":          p.Last,
		"previous":      p.Previous,
		"next":          p.Next,
		"pages":         pages,
		"current_index": p.CurrentIndex,
		"total_pages":   p.TotalPages,
	}
}

func PaginatorViews(paginators []Paginator) []map[string]any {
	out := make([]map[string]any, 0, len(paginators))
	for _, p := range paginators {
		out = append(out, PaginatorView(p))
	}
	return out
}

func ensureTrailingSlash(input string) string {
	if !strings.HasSuffix(input, "/") {
		return input + "/"
	}
	return input
}
