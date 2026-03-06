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

// TaggedLine is a log line annotated with the source that produced it.
type TaggedLine struct {
	Source string
	Line   string
}

// LogSink bridges slog JSON output via an io.Writer into a channel
// consumed by the TUI model. Each sink carries a source tag so the
// TUI can route messages to the appropriate log panel.
type LogSink struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	ch     chan TaggedLine
	writer Writer
	source string
}

// NewLogSink creates a LogSink tagged with source that also writes through to
// writer (may be nil). The source string identifies the origin of the log
// lines (e.g. "general", "serve", "api").
func NewLogSink(source string, writer Writer) *LogSink {
	return &LogSink{
		ch:     make(chan TaggedLine, 200),
		writer: writer,
		source: source,
	}
}

// Source returns the source tag for this sink.
func (s *LogSink) Source() string {
	return s.source
}

// Channel returns the read-only channel for tagged log lines.
func (s *LogSink) Channel() <-chan TaggedLine {
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
		case s.ch <- TaggedLine{Source: s.source, Line: line}:
		default:
		}
	}
	return n, nil
}

// MergeChannels fans-in multiple LogSink channels into a single read-only
// channel. The returned channel is closed when all source channels are
// drained and closed. This is useful for the TUI to listen on one channel
// that aggregates all log sources.
func MergeChannels(sinks ...*LogSink) <-chan TaggedLine {
	out := make(chan TaggedLine, 200)
	var wg sync.WaitGroup
	for _, s := range sinks {
		if s == nil {
			continue
		}
		wg.Add(1)
		ch := s.Channel()
		go func() {
			defer wg.Done()
			for tl := range ch {
				out <- tl
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
