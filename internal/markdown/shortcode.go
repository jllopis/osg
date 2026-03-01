package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// --- Block shortcodes (paired: opening + closing tag) ---

var blockShortcodeNames = []string{"note", "warning", "tip", "details", "figure", "quote", "tabs", "tab"}

var blockShortcodeHandlers = map[string]func(args, content string) string{
	"note":    func(args, content string) string { return renderAdmonition("note", "info", args, content) },
	"warning": func(args, content string) string { return renderAdmonition("warning", "warning", args, content) },
	"tip":     func(args, content string) string { return renderAdmonition("tip", "tip", args, content) },
	"details": renderDetails,
	"figure":  renderFigure,
	"quote":   renderQuote,
	"tabs":    renderTabs,
	"tab":     renderTab,
}

var blockShortcodeRegexes = buildBlockRegexes()

func buildBlockRegexes() map[string]*regexp.Regexp {
	regexes := make(map[string]*regexp.Regexp, len(blockShortcodeNames))
	for _, name := range blockShortcodeNames {
		pattern := fmt.Sprintf(`\{\{<\s*%s(?:\s+(.*?))?\s*>\}\}([\s\S]*?)\{\{<\s*/%s\s*>\}\}`, name, name)
		regexes[name] = regexp.MustCompile(pattern)
	}
	return regexes
}

// --- Inline shortcodes (self-closing: no closing tag) ---

var inlineShortcodeNames = []string{"youtube", "twitter", "codepen"}

var inlineShortcodeHandlers = map[string]func(args string) string{
	"youtube": renderYouTube,
	"twitter": renderTwitter,
	"codepen": renderCodePen,
}

var inlineShortcodeRegexes = buildInlineRegexes()

func buildInlineRegexes() map[string]*regexp.Regexp {
	regexes := make(map[string]*regexp.Regexp, len(inlineShortcodeNames))
	for _, name := range inlineShortcodeNames {
		// Match {{< name args />}} or {{< name args >}} (self-closing)
		pattern := fmt.Sprintf(`\{\{<\s*%s(?:\s+(.*?))?\s*/?\s*>\}\}`, name)
		regexes[name] = regexp.MustCompile(pattern)
	}
	return regexes
}

// ExpandShortcodes processes shortcode blocks in Markdown source and replaces
// them with HTML output. Unknown shortcodes are left unchanged.
func ExpandShortcodes(input string) string {
	result := input

	// Block shortcodes first (they have closing tags, need inner expansion)
	for _, name := range blockShortcodeNames {
		re := blockShortcodeRegexes[name]
		handler := blockShortcodeHandlers[name]
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) < 3 {
				return match
			}
			args := strings.TrimSpace(parts[1])
			content := parts[2]
			return handler(args, content)
		})
	}

	// Inline shortcodes (self-closing)
	for _, name := range inlineShortcodeNames {
		re := inlineShortcodeRegexes[name]
		handler := inlineShortcodeHandlers[name]
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) < 2 {
				return match
			}
			args := strings.TrimSpace(parts[1])
			return handler(args)
		})
	}

	return result
}

// --- Argument parsing ---

// parseArgs splits a raw argument string into key=value pairs.
// Supports: key="value", key='value', key=value, and bare positional args.
func parseArgs(raw string) map[string]string {
	out := make(map[string]string)
	if raw == "" {
		return out
	}

	re := regexp.MustCompile(`(\w+)\s*=\s*"([^"]*)"|\b(\w+)\s*=\s*'([^']*)'|\b(\w+)\s*=\s*(\S+)`)
	matches := re.FindAllStringSubmatch(raw, -1)
	for _, m := range matches {
		switch {
		case m[1] != "":
			out[m[1]] = m[2]
		case m[3] != "":
			out[m[3]] = m[4]
		case m[5] != "":
			out[m[5]] = m[6]
		}
	}

	// If no key=value found, treat entire string as a bare positional arg
	if len(out) == 0 {
		out["_pos"] = strings.Trim(raw, `"'`)
	}

	return out
}

// --- Block shortcode handlers ---

func renderAdmonition(kind string, defaultTitle string, args string, content string) string {
	title := defaultTitle
	if args != "" {
		title = strings.Trim(args, `"'`)
	}
	content = strings.TrimSpace(content)
	return fmt.Sprintf(
		`<div class="admonition admonition-%s"><p class="admonition-title">%s</p>%s</div>`,
		kind, title, "\n\n"+content+"\n\n",
	)
}

func renderDetails(args string, content string) string {
	summary := "Details"
	if args != "" {
		summary = strings.Trim(args, `"'`)
	}
	content = strings.TrimSpace(content)
	return fmt.Sprintf(
		"<details><summary>%s</summary>\n\n%s\n\n</details>",
		summary, content,
	)
}

func renderFigure(args string, content string) string {
	p := parseArgs(args)
	src := p["src"]
	if src == "" {
		src = p["_pos"]
	}
	caption := p["caption"]
	alt := p["alt"]
	if alt == "" {
		alt = caption
	}
	class := p["class"]
	width := p["width"]
	link := p["link"]

	var buf strings.Builder
	classAttr := "figure"
	if class != "" {
		classAttr += " " + class
	}
	fmt.Fprintf(&buf, `<figure class="%s">`, classAttr)

	if link != "" {
		fmt.Fprintf(&buf, `<a href="%s">`, link)
	}

	imgTag := fmt.Sprintf(`<img src="%s" alt="%s"`, src, alt)
	if width != "" {
		imgTag += fmt.Sprintf(` width="%s"`, width)
	}
	imgTag += ` loading="lazy" />`
	buf.WriteString(imgTag)

	if link != "" {
		buf.WriteString("</a>")
	}

	// Inner content (markdown) as figcaption if present, otherwise use caption arg
	inner := strings.TrimSpace(content)
	if inner != "" {
		fmt.Fprintf(&buf, "\n\n<figcaption>%s</figcaption>\n\n", inner)
	} else if caption != "" {
		fmt.Fprintf(&buf, "\n\n<figcaption>%s</figcaption>\n\n", caption)
	}

	buf.WriteString("</figure>")
	return buf.String()
}

func renderQuote(args string, content string) string {
	p := parseArgs(args)
	author := p["author"]
	if author == "" {
		author = p["_pos"]
	}
	source := p["source"]

	content = strings.TrimSpace(content)

	var buf strings.Builder
	buf.WriteString(`<blockquote class="quote">`)
	buf.WriteString("\n\n" + content + "\n\n")

	if author != "" || source != "" {
		buf.WriteString("<footer class=\"quote-attribution\">")
		if author != "" {
			fmt.Fprintf(&buf, `<cite class="quote-author">%s</cite>`, author)
		}
		if source != "" {
			if author != "" {
				buf.WriteString(", ")
			}
			fmt.Fprintf(&buf, `<span class="quote-source">%s</span>`, source)
		}
		buf.WriteString("</footer>")
	}

	buf.WriteString("</blockquote>")
	return buf.String()
}

func renderTabs(args string, content string) string {
	// Tabs wrapper: processes inner {{< tab >}} shortcodes
	content = strings.TrimSpace(content)
	return fmt.Sprintf(`<div class="tabs">%s</div>`, "\n\n"+content+"\n\n")
}

func renderTab(args string, content string) string {
	title := "Tab"
	if args != "" {
		title = strings.Trim(args, `"'`)
	}
	content = strings.TrimSpace(content)
	return fmt.Sprintf(
		`<div class="tab" data-tab-title="%s"><div class="tab-content">%s</div></div>`,
		title, "\n\n"+content+"\n\n",
	)
}

// --- Inline shortcode handlers ---

func renderYouTube(args string) string {
	id := extractVideoID(strings.Trim(args, `"'`))
	if id == "" {
		return ""
	}
	return fmt.Sprintf(
		`<div class="embed embed-youtube"><iframe src="https://www.youtube-nocookie.com/embed/%s" `+
			`frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" `+
			`allowfullscreen loading="lazy" title="YouTube video"></iframe></div>`,
		id,
	)
}

// extractVideoID extracts the YouTube video ID from a URL or bare ID.
func extractVideoID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	// Already a bare ID (11 chars, no slashes)
	if len(input) == 11 && !strings.Contains(input, "/") && !strings.Contains(input, ".") {
		return input
	}
	// youtube.com/watch?v=ID
	if re := regexp.MustCompile(`[?&]v=([a-zA-Z0-9_-]{11})`); re.MatchString(input) {
		return re.FindStringSubmatch(input)[1]
	}
	// youtu.be/ID
	if re := regexp.MustCompile(`youtu\.be/([a-zA-Z0-9_-]{11})`); re.MatchString(input) {
		return re.FindStringSubmatch(input)[1]
	}
	// youtube.com/embed/ID
	if re := regexp.MustCompile(`/embed/([a-zA-Z0-9_-]{11})`); re.MatchString(input) {
		return re.FindStringSubmatch(input)[1]
	}
	// Fallback: treat as bare ID
	return input
}

func renderTwitter(args string) string {
	url := strings.Trim(args, `"'`)
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	// Normalize x.com to twitter.com for embed compatibility
	url = strings.Replace(url, "x.com", "twitter.com", 1)
	return fmt.Sprintf(
		`<div class="embed embed-twitter"><blockquote class="twitter-tweet"><a href="%s"></a></blockquote>`+
			`<script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script></div>`,
		url,
	)
}

func renderCodePen(args string) string {
	p := parseArgs(args)
	url := p["_pos"]
	if url == "" {
		url = p["url"]
	}
	if url == "" {
		return ""
	}
	url = strings.Trim(url, `"'`)

	// Parse codepen URL: https://codepen.io/USER/pen/PENID
	re := regexp.MustCompile(`codepen\.io/([^/]+)/pen/([^/?#]+)`)
	m := re.FindStringSubmatch(url)
	if len(m) < 3 {
		return fmt.Sprintf(`<div class="embed embed-codepen"><a href="%s">View on CodePen</a></div>`, url)
	}
	user := m[1]
	penID := m[2]

	height := p["height"]
	if height == "" {
		height = "400"
	}
	theme := p["theme"]
	if theme == "" {
		theme = "dark"
	}
	defaultTab := p["tab"]
	if defaultTab == "" {
		defaultTab = "result"
	}

	return fmt.Sprintf(
		`<div class="embed embed-codepen"><iframe height="%s" scrolling="no" `+
			`src="https://codepen.io/%s/embed/%s?default-tab=%s&theme-id=%s" `+
			`frameborder="no" loading="lazy" allowtransparency="true" allowfullscreen="true" `+
			`title="CodePen"></iframe></div>`,
		height, user, penID, defaultTab, theme,
	)
}
