package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadNode reads a YAML file and returns the root document node.
// The returned node preserves comments, ordering and style.
// If the file does not exist, returns a new empty mapping node.
func LoadNode(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return an empty document with a mapping root.
			return &yaml.Node{
				Kind: yaml.DocumentNode,
				Content: []*yaml.Node{
					{Kind: yaml.MappingNode, Tag: "!!map"},
				},
			}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &doc, nil
}

// SaveNode writes a yaml.Node document to the given file path,
// preserving comments and formatting.
func SaveNode(path string, doc *yaml.Node) error {
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}

// rootMapping returns the top-level mapping node from a document node.
// If doc is already a MappingNode it is returned directly.
func rootMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	if doc.Kind == yaml.MappingNode {
		return doc
	}
	return nil
}

// GetNodeValue retrieves the string value for a dotted key path
// (e.g. "ai.provider", "interactions.comments.enabled").
// Returns the value and true if found, or "" and false if the key
// does not exist. For sequence nodes, values are joined with ", ".
func GetNodeValue(doc *yaml.Node, key string) (string, bool) {
	node := findNode(doc, key)
	if node == nil {
		return "", false
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Value, true
	case yaml.SequenceNode:
		var items []string
		for _, child := range node.Content {
			if child.Kind == yaml.ScalarNode {
				items = append(items, child.Value)
			}
		}
		return strings.Join(items, ", "), true
	default:
		return "", false
	}
}

// GetNodeSequence retrieves the raw sequence items for a dotted key path.
// Unlike GetNodeValue, this returns each element including empty strings.
func GetNodeSequence(doc *yaml.Node, key string) ([]string, bool) {
	node := findNode(doc, key)
	if node == nil {
		return nil, false
	}
	if node.Kind != yaml.SequenceNode {
		return nil, false
	}
	var items []string
	for _, child := range node.Content {
		if child.Kind == yaml.ScalarNode {
			items = append(items, child.Value)
		}
	}
	return items, true
}

// SetNodeValue sets a scalar value at the given dotted key path.
// Intermediate mapping nodes are created if they don't exist.
// Comments on existing nodes are preserved.
func SetNodeValue(doc *yaml.Node, key string, value string) {
	parts := strings.Split(key, ".")
	m := rootMapping(doc)
	if m == nil {
		return
	}

	for i, part := range parts {
		if i == len(parts)-1 {
			// Set the value on this mapping.
			setScalar(m, part, value)
			return
		}
		// Navigate or create intermediate mapping.
		m = getOrCreateMapping(m, part)
	}
}

// SetNodeSequence sets a sequence (list) value at the given dotted key path.
func SetNodeSequence(doc *yaml.Node, key string, values []string) {
	parts := strings.Split(key, ".")
	m := rootMapping(doc)
	if m == nil {
		return
	}

	for i, part := range parts {
		if i == len(parts)-1 {
			setSequence(m, part, values)
			return
		}
		m = getOrCreateMapping(m, part)
	}
}

// DeleteNodeKey removes a key from the given dotted key path.
// Does nothing if the key does not exist.
func DeleteNodeKey(doc *yaml.Node, key string) {
	parts := strings.Split(key, ".")
	m := rootMapping(doc)
	if m == nil {
		return
	}

	for i, part := range parts {
		if i == len(parts)-1 {
			deleteKey(m, part)
			return
		}
		child := findMappingChild(m, part)
		if child == nil {
			return
		}
		m = child
	}
}

// --- internal helpers ---

// findNode navigates a dotted key path and returns the value node.
func findNode(doc *yaml.Node, key string) *yaml.Node {
	parts := strings.Split(key, ".")
	m := rootMapping(doc)
	if m == nil {
		return nil
	}

	for i, part := range parts {
		idx := findKeyIndex(m, part)
		if idx < 0 {
			return nil
		}
		valueNode := m.Content[idx+1]
		if i == len(parts)-1 {
			return valueNode
		}
		if valueNode.Kind != yaml.MappingNode {
			return nil
		}
		m = valueNode
	}
	return nil
}

// findKeyIndex returns the index of the key node in a mapping,
// or -1 if not found.
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

// findMappingChild returns the value node for a key in a mapping,
// but only if it is itself a mapping. Returns nil otherwise.
func findMappingChild(m *yaml.Node, key string) *yaml.Node {
	idx := findKeyIndex(m, key)
	if idx < 0 {
		return nil
	}
	child := m.Content[idx+1]
	if child.Kind == yaml.MappingNode {
		return child
	}
	return nil
}

// setScalar sets or creates a scalar key-value pair in a mapping node.
func setScalar(m *yaml.Node, key string, value string) {
	idx := findKeyIndex(m, key)
	if idx >= 0 {
		// Preserve the existing value node's comments by only updating the value.
		vn := m.Content[idx+1]
		vn.Kind = yaml.ScalarNode
		vn.Tag = guessScalarTag(value)
		vn.Value = value
		vn.Content = nil
		return
	}
	// Append new key-value pair.
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: guessScalarTag(value), Value: value},
	)
}

// setSequence sets or creates a sequence key-value pair in a mapping node.
func setSequence(m *yaml.Node, key string, values []string) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, v := range values {
		seq.Content = append(seq.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   guessScalarTag(v),
			Value: v,
		})
	}

	idx := findKeyIndex(m, key)
	if idx >= 0 {
		// Replace the value node but keep the key node (preserving its comments).
		oldValue := m.Content[idx+1]
		seq.HeadComment = oldValue.HeadComment
		seq.LineComment = oldValue.LineComment
		seq.FootComment = oldValue.FootComment
		m.Content[idx+1] = seq
		return
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		seq,
	)
}

// getOrCreateMapping navigates to a child mapping under key, creating
// it if it doesn't exist.
func getOrCreateMapping(m *yaml.Node, key string) *yaml.Node {
	idx := findKeyIndex(m, key)
	if idx >= 0 {
		child := m.Content[idx+1]
		if child.Kind == yaml.MappingNode {
			return child
		}
		// Overwrite non-mapping with a new mapping (unusual case).
		newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		m.Content[idx+1] = newMap
		return newMap
	}
	// Create new mapping.
	newMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		newMap,
	)
	return newMap
}

// deleteKey removes a key-value pair from a mapping node.
func deleteKey(m *yaml.Node, key string) {
	idx := findKeyIndex(m, key)
	if idx < 0 {
		return
	}
	// Remove the key node and value node (two consecutive elements).
	m.Content = append(m.Content[:idx], m.Content[idx+2:]...)
}

// guessScalarTag returns the appropriate YAML tag for a value string.
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
