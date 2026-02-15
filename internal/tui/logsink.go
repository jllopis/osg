package tui

import (
	"bytes"
	"strings"
	"sync"
)

// Writer is the interface used to write history lines (satisfied by History).
type Writer interface {
	Write([]byte) (int, error)
}

// LogSink bridges slog JSON output via an io.Writer into a channel
// consumed by the TUI model.
type LogSink struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	ch     chan string
	writer Writer
}

// NewLogSink creates a LogSink that also writes through to writer (may be nil).
func NewLogSink(writer Writer) *LogSink {
	return &LogSink{
		ch:     make(chan string, 200),
		writer: writer,
	}
}

// Channel returns the read-only channel for log lines.
func (s *LogSink) Channel() <-chan string {
	return s.ch
}

// Write implements io.Writer. It buffers input and flushes complete lines
// (delimited by '\n') to both the writer and the channel.
func (s *LogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, _ := s.buf.Write(p)
	for {
		data := s.buf.Bytes()
		idx := bytes.IndexByte(data, '\n')
		if idx == -1 {
			break
		}
		line := strings.TrimRight(string(data[:idx]), "\r")
		_ = s.buf.Next(idx + 1)
		if strings.TrimSpace(line) == "" {
			continue
		}
		if s.writer != nil {
			_, _ = s.writer.Write([]byte(line + "\n"))
		}
		select {
		case s.ch <- line:
		default:
		}
	}
	return n, nil
}
