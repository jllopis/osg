package publish

import "strings"

// GetOSGBlock extracts the "osg" map from frontmatter, returning nil if absent or wrong type.
// It handles both map format and list-of-maps format (YAML with - prefixes).
func GetOSGBlock(fm map[string]any) map[string]any {
	if fm == nil {
		return nil
	}
	raw, ok := fm["osg"]
	if !ok {
		return nil
	}

	// Direct map format: osg:\n  publish: true
	if m, ok := raw.(map[string]any); ok {
		return m
	}

	// List-of-maps format: osg:\n  - publish: true\n  - image: "..."
	// Merge all list items into a single map.
	if list, ok := raw.([]any); ok {
		merged := make(map[string]any)
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				for k, v := range m {
					merged[k] = v
				}
			}
		}
		if len(merged) > 0 {
			return merged
		}
	}

	return nil
}

func ShouldPublish(fm map[string]any) (publish bool, draft bool) {
	if fm == nil {
		return false, false
	}

	// Check osg.publish first (takes precedence)
	if osg := GetOSGBlock(fm); osg != nil {
		if val, ok := osg["publish"]; ok {
			return evalPublishValue(val)
		}
	}

	// Fallback to top-level publish
	val, ok := fm["publish"]
	if !ok {
		return false, false
	}

	return evalPublishValue(val)
}

// GetOSGString extracts a string value from the "osg" block in frontmatter.
// Returns empty string if the key is absent, the osg block is missing, or the value is not a string.
func GetOSGString(fm map[string]any, key string) string {
	osg := GetOSGBlock(fm)
	if osg == nil {
		return ""
	}
	val, ok := osg[key]
	if !ok {
		return ""
	}
	s, ok := val.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// GetOSGBool extracts a boolean value from the "osg" block in frontmatter.
// Returns false if the key is absent, the osg block is missing, or the value is not a bool/string.
func GetOSGBool(fm map[string]any, key string) bool {
	osg := GetOSGBlock(fm)
	if osg == nil {
		return false
	}
	val, ok := osg[key]
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

func evalPublishValue(val any) (publish bool, draft bool) {
	switch v := val.(type) {
	case bool:
		if v {
			return true, false
		}
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		if lower == "true" {
			return true, false
		}
		if lower == "draft" {
			return true, true
		}
	}
	return false, false
}
