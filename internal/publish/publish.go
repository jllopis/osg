package publish

import "strings"

// GetOSGBlock extracts the "osg" map from frontmatter, returning nil if absent or wrong type.
func GetOSGBlock(fm map[string]any) map[string]any {
	if fm == nil {
		return nil
	}
	raw, ok := fm["osg"]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return m
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
