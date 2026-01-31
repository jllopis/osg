package publish

import "strings"

func ShouldPublish(fm map[string]any) (publish bool, draft bool) {
	if fm == nil {
		return false, false
	}

	val, ok := fm["publish"]
	if !ok {
		return false, false
	}

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
