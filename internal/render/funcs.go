package render

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"

	"osg/internal/markdown"
	"osg/internal/site"
	"osg/internal/slug"
	"osg/internal/taxonomy"

	"osg/internal/i18n"
	imgopt "osg/internal/image"

	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

type Context struct {
	BaseURL         string
	ContentDir      string
	StaticDir       string
	PublicDir       string
	DefaultLanguage string
	Site            *site.Site
	Taxonomies      map[string]*taxonomy.Index
	ImageResults    map[string]*imgopt.Result
	I18n            *i18n.Bundle
}

func FuncMap(ctx Context) template.FuncMap {
	pageIndex := map[string]*site.Page{}
	sectionIndex := map[string]*site.Section{}
	if ctx.Site != nil {
		for _, page := range ctx.Site.Pages {
			pageIndex[normalizePath(page.Path)] = page
		}
		for _, section := range ctx.Site.Sections {
			sectionIndex[normalizePath(section.Path)] = section
		}
	}

	return template.FuncMap{
		"markdown":           markdownFilter,
		"base64_encode":      base64Encode,
		"base64_decode":      base64Decode,
		"regex_replace":      regexReplace,
		"num_format":         numFormat,
		"get_page":           getPageFunc(pageIndex),
		"get_section":        getSectionFunc(sectionIndex),
		"get_taxonomy_url":   getTaxonomyURLFunc(ctx),
		"get_taxonomy":       getTaxonomyFunc(ctx),
		"get_url":            getURLFunc(ctx),
		"get_hash":           getHashFunc(ctx),
		"get_image_metadata": getImageMetadataFunc(ctx),
		"load_data":          loadDataFunc(ctx),
		"trans":              transFunc(ctx),
		"date_format":        dateFormatFunc(ctx),
		"picture":            pictureFunc(ctx),
	}
}

func markdownFilter(input any) (template.HTML, error) {
	if input == nil {
		return "", nil
	}

	switch v := input.(type) {
	case string:
		out, err := markdown.RenderString(v)
		return template.HTML(out), err
	case []byte:
		out, err := markdown.Render(v)
		return template.HTML(out), err
	default:
		return "", fmt.Errorf("markdown: unsupported type %T", input)
	}
}

func base64Encode(input any) (string, error) {
	if input == nil {
		return "", nil
	}

	switch v := input.(type) {
	case string:
		return base64.StdEncoding.EncodeToString([]byte(v)), nil
	case []byte:
		return base64.StdEncoding.EncodeToString(v), nil
	default:
		return "", fmt.Errorf("base64_encode: unsupported type %T", input)
	}
}

func base64Decode(input any) (string, error) {
	if input == nil {
		return "", nil
	}

	s, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("base64_decode: unsupported type %T", input)
	}

	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func regexReplace(input any, pattern string, repl string) (string, error) {
	if input == nil {
		return "", nil
	}

	s, ok := input.(string)
	if !ok {
		return "", fmt.Errorf("regex_replace: unsupported type %T", input)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	return re.ReplaceAllString(s, repl), nil
}

func numFormat(input any, locale string) (string, error) {
	if input == nil {
		return "", nil
	}

	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}

	printer := message.NewPrinter(language.Make(locale))

	switch v := input.(type) {
	case int:
		return printer.Sprintf("%d", v), nil
	case int64:
		return printer.Sprintf("%d", v), nil
	case float32:
		return printer.Sprintf("%g", v), nil
	case float64:
		return printer.Sprintf("%g", v), nil
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return "", nil
		}
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return printer.Sprintf("%d", i), nil
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return printer.Sprintf("%g", f), nil
		}
		return v, nil
	default:
		return fmt.Sprint(input), nil
	}
}

func getPageFunc(index map[string]*site.Page) func(string, ...string) map[string]any {
	return func(path string, _ ...string) map[string]any {
		path = normalizePath(path)
		page, ok := index[path]
		if !ok {
			return nil
		}
		return page.View()
	}
}

func getSectionFunc(index map[string]*site.Section) func(string, ...bool) map[string]any {
	return func(path string, metadataOnly ...bool) map[string]any {
		path = normalizePath(path)
		section, ok := index[path]
		if !ok {
			return nil
		}
		view := section.View()
		if len(metadataOnly) > 0 && metadataOnly[0] {
			view["pages"] = []map[string]any{}
			view["subsections"] = []map[string]any{}
		}
		return view
	}
}

func getTaxonomyURLFunc(ctx Context) func(string, string, ...string) string {
	return func(kind string, name string, _ ...string) string {
		index, ok := ctx.Taxonomies[kind]
		if !ok {
			return ""
		}
		termSlug := slug.Slugify(name)
		if term, ok := index.Terms[termSlug]; ok {
			return term.Permalink
		}
		return buildURL(ctx.BaseURL, "/"+kind+"/"+termSlug+"/")
	}
}

func getTaxonomyFunc(ctx Context) func(string) map[string]any {
	return func(kind string) map[string]any {
		index, ok := ctx.Taxonomies[kind]
		if !ok {
			return nil
		}
		return map[string]any{
			"taxonomy": taxonomy.ConfigView(index.Config),
			"terms":    taxonomy.TermViews(index.TermsSorted()),
		}
	}
}

func getURLFunc(ctx Context) func(string, ...any) string {
	return func(path string, opts ...any) string {
		trailing := boolArg(opts, 0, false)
		cachebust := boolArg(opts, 1, false)

		url := buildURL(ctx.BaseURL, path)
		if trailing {
			url = ensureTrailingSlash(url)
		}
		if !cachebust {
			return url
		}

		hash, err := hashForPath(ctx, path, "sha256", true)
		if err != nil || hash == "" {
			return url
		}
		separator := "?"
		if strings.Contains(url, "?") {
			separator = "&"
		}
		return url + separator + "v=" + hash
	}
}

func getHashFunc(ctx Context) func(string, ...any) (string, error) {
	return func(input string, args ...any) (string, error) {
		algo := stringArg(args, 0, "sha256")
		base64Out := boolArg(args, 1, false)
		return hashForPath(ctx, input, algo, base64Out)
	}
}

func getImageMetadataFunc(ctx Context) func(string, ...any) (map[string]any, error) {
	return func(input string, args ...any) (map[string]any, error) {
		allowMissing := boolArg(args, 0, false)
		path, err := resolveFilePath(ctx, input)
		if err != nil {
			if allowMissing {
				return map[string]any{}, nil
			}
			return nil, err
		}

		file, err := os.Open(path)
		if err != nil {
			if allowMissing {
				return map[string]any{}, nil
			}
			return nil, err
		}
		defer func() { _ = file.Close() }()

		cfg, format, err := image.DecodeConfig(file)
		if err != nil {
			if allowMissing {
				return map[string]any{}, nil
			}
			return nil, err
		}

		mimeType := mime.TypeByExtension("." + format)
		if mimeType == "" {
			mimeType = "image/" + format
		}

		return map[string]any{
			"width":  cfg.Width,
			"height": cfg.Height,
			"format": format,
			"mime":   mimeType,
		}, nil
	}
}

func loadDataFunc(ctx Context) func(string, ...any) (any, error) {
	return func(input string, args ...any) (any, error) {
		format := stringArg(args, 0, "")
		required := boolArg(args, 1, false)

		data, err := readInput(ctx, input)
		if err != nil {
			if required {
				return nil, err
			}
			return nil, nil
		}

		format = detectFormat(input, format)
		switch format {
		case "json":
			var out any
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		case "yaml", "yml":
			var out any
			if err := yaml.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		case "toml":
			var out any
			if err := toml.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		case "csv":
			return parseCSV(data)
		case "xml":
			return parseXML(data)
		default:
			return string(data), nil
		}
	}
}

// transFunc returns a template function that translates keys using the i18n
// bundle. It uses the page's language (from the template context) when
// available, falling back to the site's default language.
//
// Usage in templates:
//
//	{{ trans "key" }}             — uses the current page/default language
//	{{ trans "key" "en" }}        — explicit language override
func transFunc(ctx Context) func(string, ...string) string {
	return func(key string, lang ...string) string {
		if ctx.I18n == nil {
			return key
		}
		return ctx.I18n.Trans(key, lang...)
	}
}

// dateFormatFunc returns a template function that formats dates with
// locale-aware month names.
//
// Usage in templates:
//
//	{{ date_format .page.date "January 2, 2006" }}       — uses default language
//	{{ date_format .page.date "January 2, 2006" "en" }}  — explicit language
func dateFormatFunc(ctx Context) func(time.Time, string, ...string) string {
	return func(t time.Time, layout string, lang ...string) string {
		l := ctx.DefaultLanguage
		if len(lang) > 0 && strings.TrimSpace(lang[0]) != "" {
			l = lang[0]
		}
		return i18n.DateFormat(t, layout, l)
	}
}

// pictureFunc returns a template function that generates <picture> elements
// with responsive srcset when optimized image variants are available, or
// a plain <img> tag otherwise.
//
// Usage in templates:
//
//	{{ picture .image .title "eager" }}
//	{{ picture .image .title "lazy" }}
//	{{ picture .image .title }}          {{/* defaults to lazy */}}
func pictureFunc(ctx Context) func(string, ...string) template.HTML {
	return func(src string, args ...string) template.HTML {
		alt := ""
		loading := "lazy"
		if len(args) > 0 {
			alt = args[0]
		}
		if len(args) > 1 && args[1] != "" {
			loading = args[1]
		}
		return template.HTML(imgopt.PictureHTML(src, alt, loading, ctx.ImageResults))
	}
}

func readInput(ctx Context, input string) ([]byte, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		resp, err := http.Get(input)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("http status: %s", resp.Status)
		}
		return io.ReadAll(resp.Body)
	}

	path, err := resolveFilePath(ctx, input)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func resolveFilePath(ctx Context, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty path")
	}

	candidates := []string{}
	if filepath.IsAbs(input) {
		candidates = append(candidates, input)
	}

	trimmed := strings.TrimPrefix(input, "/")
	if ctx.PublicDir != "" {
		candidates = append(candidates, filepath.Join(ctx.PublicDir, trimmed))
	}
	if ctx.StaticDir != "" {
		candidates = append(candidates, filepath.Join(ctx.StaticDir, trimmed))
	}
	if ctx.ContentDir != "" {
		candidates = append(candidates, filepath.Join(ctx.ContentDir, trimmed))
	}
	candidates = append(candidates, input)

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("file not found: %s", input)
}

func detectFormat(input string, override string) string {
	override = strings.TrimSpace(strings.ToLower(override))
	if override != "" {
		return override
	}

	ext := strings.ToLower(filepath.Ext(input))
	if ext == "" {
		return ""
	}
	return strings.TrimPrefix(ext, ".")
}

func parseCSV(data []byte) (any, error) {
	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []map[string]string{}, nil
	}
	if len(records) == 1 {
		return []map[string]string{}, nil
	}

	headers := records[0]
	out := make([]map[string]string, 0, len(records)-1)
	for _, row := range records[1:] {
		entry := map[string]string{}
		for i, header := range headers {
			if i < len(row) {
				entry[header] = row[i]
			}
		}
		out = append(out, entry)
	}
	return out, nil
}

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Content string     `xml:",chardata"`
	Nodes   []xmlNode  `xml:",any"`
}

func parseXML(data []byte) (any, error) {
	var node xmlNode
	if err := xml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	return node, nil
}

func hashForPath(ctx Context, input string, algo string, base64Out bool) (string, error) {
	data, err := readInput(ctx, input)
	if err != nil {
		data = []byte(input)
	}

	hashed, err := hashBytes(data, algo)
	if err != nil {
		return "", err
	}

	if base64Out {
		return base64.StdEncoding.EncodeToString(hashed), nil
	}

	return fmt.Sprintf("%x", hashed), nil
}

func hashBytes(data []byte, algo string) ([]byte, error) {
	algo = strings.ToLower(strings.TrimSpace(algo))
	if algo == "" {
		algo = "sha256"
	}

	switch algo {
	case "sha1":
		h := sha1.Sum(data)
		return h[:], nil
	case "md5":
		h := md5.Sum(data)
		return h[:], nil
	case "sha256":
		fallthrough
	default:
		h := sha256.Sum256(data)
		return h[:], nil
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func buildURL(baseURL string, path string) string {
	if strings.TrimSpace(baseURL) == "" {
		return path
	}
	return strings.TrimRight(baseURL, "/") + path
}

func ensureTrailingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}
	return path + "/"
}

func boolArg(args []any, index int, fallback bool) bool {
	if len(args) <= index {
		return fallback
	}
	val := args[index]
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return fallback
	}
}

func stringArg(args []any, index int, fallback string) string {
	if len(args) <= index {
		return fallback
	}
	val := args[index]
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return fallback
	}
}
