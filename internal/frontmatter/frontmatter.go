package frontmatter

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	delimiter    = []byte("---")
	delimiterAlt = []byte("...")
)

// SplitRaw is a variant of SplitFrontmatter that returns the raw YAML
// bytes between the delimiters instead of a parsed map. It is the
// foundation for editing helpers that want to preserve comments and
// ordering by feeding the raw block straight into yaml.v3's Node API.
// The returned yamlBytes do NOT include the surrounding `---` lines.
func SplitRaw(data []byte) (yamlBytes []byte, body []byte, present bool, err error) {
	if len(data) == 0 {
		return nil, data, false, nil
	}

	lineEnd := bytes.IndexByte(data, '\n')
	if lineEnd == -1 {
		lineEnd = len(data)
	}
	firstLine := bytes.TrimSpace(bytes.TrimRight(data[:lineEnd], "\r"))
	if !bytes.Equal(firstLine, delimiter) {
		return nil, data, false, nil
	}

	cursor := lineEnd + 1
	for cursor <= len(data) {
		nextLineEnd := bytes.IndexByte(data[cursor:], '\n')
		var line []byte
		var next int
		if nextLineEnd == -1 {
			line = data[cursor:]
			next = len(data)
		} else {
			line = data[cursor : cursor+nextLineEnd]
			next = cursor + nextLineEnd + 1
		}
		trimmed := bytes.TrimSpace(bytes.TrimRight(line, "\r"))
		if bytes.Equal(trimmed, delimiter) || bytes.Equal(trimmed, delimiterAlt) {
			return data[lineEnd+1 : cursor], data[next:], true, nil
		}
		if next >= len(data) {
			break
		}
		cursor = next
	}
	return nil, nil, true, fmt.Errorf("frontmatter delimiter not closed")
}

// UpdateField sets (or deletes when value=="") a dotted key inside the
// frontmatter block of the given markdown file, preserving every other
// field, comment, and the body verbatim. Intermediate mappings are
// created as needed (e.g. setting "osg.summary" creates the `osg:`
// mapping if absent).
//
// When the file has no frontmatter at all, a new block is prepended.
// The yaml.v3 Node API is used throughout so user comments survive.
func UpdateField(data []byte, key string, value string) ([]byte, error) {
	yamlBytes, body, present, err := SplitRaw(data)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if present && len(bytes.TrimSpace(yamlBytes)) > 0 {
		if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
			return nil, fmt.Errorf("parse yaml: %w", err)
		}
	} else {
		doc = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode, Tag: "!!map"},
			},
		}
	}

	if strings.TrimSpace(value) == "" {
		deleteDotted(&doc, key)
	} else {
		setDottedScalar(&doc, key, value)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(out)
	buf.WriteString("---\n")
	// body is the original document when no frontmatter was present
	// and the post-`---` content otherwise — write either case as-is.
	buf.Write(body)
	return buf.Bytes(), nil
}

// --- internal helpers (yaml.Node manipulation) ---

func rootMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

func findKeyIndex(m *yaml.Node, key string) int {
	if m.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i < len(m.Content)-1; i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func getOrCreateMapping(m *yaml.Node, key string) *yaml.Node {
	idx := findKeyIndex(m, key)
	if idx >= 0 {
		child := m.Content[idx+1]
		if child.Kind == yaml.MappingNode {
			return child
		}
		newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		m.Content[idx+1] = newMap
		return newMap
	}
	newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		newMap,
	)
	return newMap
}

func setDottedScalar(doc *yaml.Node, key, value string) {
	parts := strings.Split(key, ".")
	m := rootMapping(doc)
	if m == nil {
		return
	}
	for i, part := range parts {
		if i == len(parts)-1 {
			idx := findKeyIndex(m, part)
			if idx >= 0 {
				vn := m.Content[idx+1]
				vn.Kind = yaml.ScalarNode
				vn.Tag = guessScalarTag(value)
				vn.Style = pickScalarStyle(value)
				vn.Value = value
				vn.Content = nil
				return
			}
			m.Content = append(m.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part},
				&yaml.Node{Kind: yaml.ScalarNode, Tag: guessScalarTag(value), Style: pickScalarStyle(value), Value: value},
			)
			return
		}
		m = getOrCreateMapping(m, part)
	}
}

func deleteDotted(doc *yaml.Node, key string) {
	parts := strings.Split(key, ".")
	m := rootMapping(doc)
	if m == nil {
		return
	}
	for i, part := range parts {
		idx := findKeyIndex(m, part)
		if idx < 0 {
			return
		}
		if i == len(parts)-1 {
			m.Content = append(m.Content[:idx], m.Content[idx+2:]...)
			return
		}
		child := m.Content[idx+1]
		if child.Kind != yaml.MappingNode {
			return
		}
		m = child
	}
}

func guessScalarTag(value string) string {
	if value == "true" || value == "false" {
		return "!!bool"
	}
	if _, err := strconv.Atoi(value); err == nil {
		return "!!int"
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "!!float"
	}
	return "!!str"
}

// pickScalarStyle keeps multi-line strings (anything with a newline)
// in literal block style so summaries with sentence breaks stay
// readable in the source file. Single-line values use the default
// style which yaml.Marshal picks (usually plain or double-quoted).
func pickScalarStyle(value string) yaml.Style {
	if strings.ContainsRune(value, '\n') {
		return yaml.LiteralStyle
	}
	return 0
}

func SplitFrontmatter(data []byte) (map[string]any, []byte, bool, error) {
	if len(data) == 0 {
		return map[string]any{}, data, false, nil
	}

	lineEnd := bytes.IndexByte(data, '\n')
	if lineEnd == -1 {
		lineEnd = len(data)
	}

	firstLine := bytes.TrimSpace(bytes.TrimRight(data[:lineEnd], "\r"))
	if !bytes.Equal(firstLine, delimiter) {
		return map[string]any{}, data, false, nil
	}

	cursor := lineEnd + 1
	for cursor <= len(data) {
		nextLineEnd := bytes.IndexByte(data[cursor:], '\n')
		var line []byte
		var next int
		if nextLineEnd == -1 {
			line = data[cursor:]
			next = len(data)
		} else {
			line = data[cursor : cursor+nextLineEnd]
			next = cursor + nextLineEnd + 1
		}

		trimmed := bytes.TrimSpace(bytes.TrimRight(line, "\r"))
		if bytes.Equal(trimmed, delimiter) || bytes.Equal(trimmed, delimiterAlt) {
			frontmatterBytes := data[lineEnd+1 : cursor]
			body := data[next:]

			fm := map[string]any{}
			if len(bytes.TrimSpace(frontmatterBytes)) > 0 {
				if err := yaml.Unmarshal(frontmatterBytes, &fm); err != nil {
					return nil, nil, true, fmt.Errorf("parse yaml: %w", err)
				}
			}

			return fm, body, true, nil
		}

		if next >= len(data) {
			break
		}
		cursor = next
	}

	return nil, nil, true, fmt.Errorf("frontmatter delimiter not closed")
}
