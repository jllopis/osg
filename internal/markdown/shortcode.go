package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// shortcodeNames lists known shortcode names.
var shortcodeNames = []string{"note", "warning", "tip", "details"}

// shortcodeHandlers maps shortcode names to their rendering functions.
var shortcodeHandlers = map[string]func(args, content string) string{
	"note":    func(args, content string) string { return renderAdmonition("note", "info", args, content) },
	"warning": func(args, content string) string { return renderAdmonition("warning", "warning", args, content) },
	"tip":     func(args, content string) string { return renderAdmonition("tip", "tip", args, content) },
	"details": renderDetails,
}

// shortcodeRegexes caches compiled regexes for each shortcode name.
var shortcodeRegexes = buildShortcodeRegexes()

func buildShortcodeRegexes() map[string]*regexp.Regexp {
	regexes := make(map[string]*regexp.Regexp, len(shortcodeNames))
	for _, name := range shortcodeNames {
		// Match {{< name [args] >}}content{{< /name >}}
		pattern := fmt.Sprintf(`\{\{<\s*%s(?:\s+(.*?))?\s*>\}\}([\s\S]*?)\{\{<\s*/%s\s*>\}\}`, name, name)
		regexes[name] = regexp.MustCompile(pattern)
	}
	return regexes
}

// ExpandShortcodes processes shortcode blocks in Markdown source and replaces
// them with HTML output. Unknown shortcodes are left unchanged.
func ExpandShortcodes(input string) string {
	result := input
	for _, name := range shortcodeNames {
		re := shortcodeRegexes[name]
		handler := shortcodeHandlers[name]
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
	return result
}

// renderAdmonition generates a styled callout block.
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

// renderDetails generates a collapsible <details> element.
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
