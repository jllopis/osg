package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type History struct {
	path string
	file *os.File
	mu   sync.Mutex
}

func NewHistory() (*History, error) {
	path := defaultHistoryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &History{path: path, file: file}, nil
}

func (h *History) Path() string {
	if h == nil {
		return ""
	}
	return h.path
}

func (h *History) Write(p []byte) (int, error) {
	if h == nil || h.file == nil {
		return len(p), nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.file.Write(p)
}

func (h *History) Append(label string, text string) {
	if h == nil {
		return
	}
	entry := fmt.Sprintf("%s [%s] %s\n", time.Now().Format(time.RFC3339), label, text)
	_, _ = h.Write([]byte(entry))
}

func (h *History) Close() {
	if h == nil || h.file == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	_ = h.file.Close()
}

func defaultHistoryPath() string {
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".osg", "tui.log")
	}
	return "osg-tui.log"
}
