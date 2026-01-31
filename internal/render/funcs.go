package render

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"html/template"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"osg/internal/markdown"
)

func FuncMap() map[string]any {
	return map[string]any{
		"markdown":      markdownFilter,
		"base64_encode": base64Encode,
		"base64_decode": base64Decode,
		"regex_replace": regexReplace,
		"num_format":    numFormat,
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
