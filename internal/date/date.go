package date

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var candidateKeys = []string{
	"date",
	"created",
	"created_at",
	"createdAt",
	"updated",
	"modified",
	"lastmod",
}

var layouts = []string{
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02",
}

func Derive(fm map[string]any, info os.FileInfo) time.Time {
	if fm != nil {
		for _, key := range candidateKeys {
			if val, ok := fm[key]; ok {
				if t, ok := Parse(val); ok {
					return t
				}
			}
		}
	}
	return info.ModTime()
}

func FormatISO(t time.Time) string {
	return t.Format("2006-01-02")
}

func FormatPath(t time.Time) string {
	return fmt.Sprintf("%04d/%02d/%02d", t.Year(), t.Month(), t.Day())
}

func Parse(val any) (time.Time, bool) {
	switch v := val.(type) {
	case time.Time:
		return v, true
	case int64:
		return time.Unix(v, 0), true
	case int:
		return time.Unix(int64(v), 0), true
	case float64:
		return time.Unix(int64(v), 0), true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return time.Time{}, false
		}
		for _, layout := range layouts {
			if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}
