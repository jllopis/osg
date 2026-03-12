package importer

import (
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
	"time"
)

// WXR is the top-level WordPress eXtended RSS structure.
type wxrRSS struct {
	Channel wxrChannel `xml:"channel"`
}

type wxrChannel struct {
	Items []wxrItem `xml:"item"`
}

type wxrItem struct {
	Title      string        `xml:"title"`
	Link       string        `xml:"link"`
	PubDate    string        `xml:"pubDate"`
	PostDate   string        `xml:"http://wordpress.org/export/ post_date"`
	PostName   string        `xml:"http://wordpress.org/export/ post_name"`
	PostType   string        `xml:"http://wordpress.org/export/ post_type"`
	Status     string        `xml:"http://wordpress.org/export/ status"`
	Content    string        `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Categories []wxrCategory `xml:"category"`
}

type wxrCategory struct {
	Domain string `xml:"domain,attr"`
	Name   string `xml:",chardata"`
}

// ParseWordpress reads a WordPress WXR export XML file and returns posts.
func ParseWordpress(path string) ([]Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wordpress export: %w", err)
	}

	var rss wxrRSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		return nil, fmt.Errorf("parse wordpress XML: %w", err)
	}

	var posts []Post
	for _, item := range rss.Channel.Items {
		if item.PostType != "" && item.PostType != "post" {
			continue
		}

		date := parseWXRDate(item.PostDate, item.PubDate)
		draft := item.Status == "draft"

		var tags, categories []string
		for _, cat := range item.Categories {
			switch cat.Domain {
			case "post_tag":
				tags = append(tags, cat.Name)
			case "category":
				categories = append(categories, cat.Name)
			}
		}

		content := htmlToMarkdown(item.Content)

		posts = append(posts, Post{
			Title:      item.Title,
			Date:       date,
			Tags:       tags,
			Categories: categories,
			Content:    content,
			Slug:       item.PostName,
			Draft:      draft,
		})
	}

	return posts, nil
}

func parseWXRDate(postDate, pubDate string) time.Time {
	if postDate != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", postDate); err == nil {
			return t
		}
	}
	if pubDate != "" {
		if t, err := time.Parse(time.RFC1123Z, pubDate); err == nil {
			return t
		}
		if t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", pubDate); err == nil {
			return t
		}
	}
	return time.Now()
}

var (
	reHTMLTag    = regexp.MustCompile(`<[^>]+>`)
	reMultiLine  = regexp.MustCompile(`\n{3,}`)
	reHref       = regexp.MustCompile(`<a\s[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	reImg        = regexp.MustCompile(`<img\s[^>]*src="([^"]*)"[^>]*(?:alt="([^"]*)")?[^>]*/?>`)
	reStrong     = regexp.MustCompile(`<(?:strong|b)>(.*?)</(?:strong|b)>`)
	reEm         = regexp.MustCompile(`<(?:em|i)>(.*?)</(?:em|i)>`)
	reH1         = regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	reH2         = regexp.MustCompile(`<h2[^>]*>(.*?)</h2>`)
	reH3         = regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`)
	reLi         = regexp.MustCompile(`<li[^>]*>(.*?)</li>`)
	reBlockquote = regexp.MustCompile(`<blockquote[^>]*>(.*?)</blockquote>`)
	rePre        = regexp.MustCompile(`<pre[^>]*>(.*?)</pre>`)
	reCode       = regexp.MustCompile(`<code[^>]*>(.*?)</code>`)
	reBr         = regexp.MustCompile(`<br\s*/?>`)
	reParagraph  = regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
)

// htmlToMarkdown does a basic HTML-to-Markdown conversion.
func htmlToMarkdown(s string) string {
	// Preserve line breaks.
	s = reBr.ReplaceAllString(s, "\n")

	// Convert structural elements.
	s = reH1.ReplaceAllString(s, "\n# $1\n")
	s = reH2.ReplaceAllString(s, "\n## $1\n")
	s = reH3.ReplaceAllString(s, "\n### $1\n")
	s = reBlockquote.ReplaceAllString(s, "\n> $1\n")
	s = rePre.ReplaceAllString(s, "\n```\n$1\n```\n")
	s = reCode.ReplaceAllString(s, "`$1`")
	s = reLi.ReplaceAllString(s, "- $1\n")

	// Convert inline elements.
	s = reHref.ReplaceAllString(s, "[$2]($1)")
	s = reImg.ReplaceAllString(s, "![$2]($1)")
	s = reStrong.ReplaceAllString(s, "**$1**")
	s = reEm.ReplaceAllString(s, "*$1*")

	// Paragraphs.
	s = reParagraph.ReplaceAllString(s, "\n$1\n")

	// Strip remaining HTML tags.
	s = reHTMLTag.ReplaceAllString(s, "")

	// Decode HTML entities.
	s = html.UnescapeString(s)

	// Clean up excessive newlines.
	s = reMultiLine.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
