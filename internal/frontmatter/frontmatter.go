package frontmatter

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	delimiter    = []byte("---")
	delimiterAlt = []byte("...")
)

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
