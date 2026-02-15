package tui

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// NewLogSink
// ---------------------------------------------------------------------------

func TestNewLogSink(t *testing.T) {
	t.Run("creates with nil writer", func(t *testing.T) {
		sink := NewLogSink(nil)
		if sink == nil {
			t.Fatal("NewLogSink(nil) returned nil")
		}
		ch := sink.Channel()
		if ch == nil {
			t.Fatal("Channel() returned nil")
		}
	})

	t.Run("creates with writer", func(t *testing.T) {
		w := &captureWriter{}
		sink := NewLogSink(w)
		if sink == nil {
			t.Fatal("NewLogSink returned nil")
		}
	})
}

// ---------------------------------------------------------------------------
// LogSink.Write
// ---------------------------------------------------------------------------

func TestLogSinkWrite(t *testing.T) {
	t.Run("complete line", func(t *testing.T) {
		sink := NewLogSink(nil)
		n, err := sink.Write([]byte("hello world\n"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 12 {
			t.Errorf("n = %d; want 12", n)
		}

		select {
		case line := <-sink.Channel():
			if line != "hello world" {
				t.Errorf("line = %q; want \"hello world\"", line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}
	})

	t.Run("partial writes buffered", func(t *testing.T) {
		sink := NewLogSink(nil)
		sink.Write([]byte("hel"))
		sink.Write([]byte("lo\n"))

		select {
		case line := <-sink.Channel():
			if line != "hello" {
				t.Errorf("line = %q; want \"hello\"", line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}
	})

	t.Run("multiple lines in one write", func(t *testing.T) {
		sink := NewLogSink(nil)
		sink.Write([]byte("line1\nline2\n"))

		select {
		case line := <-sink.Channel():
			if line != "line1" {
				t.Errorf("line1 = %q", line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out for line1")
		}

		select {
		case line := <-sink.Channel():
			if line != "line2" {
				t.Errorf("line2 = %q", line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out for line2")
		}
	})

	t.Run("blank lines skipped", func(t *testing.T) {
		sink := NewLogSink(nil)
		sink.Write([]byte("\n\nhello\n\n"))

		select {
		case line := <-sink.Channel():
			if line != "hello" {
				t.Errorf("line = %q; want \"hello\"", line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}

		// Ensure no more lines.
		select {
		case line := <-sink.Channel():
			t.Errorf("unexpected extra line: %q", line)
		case <-time.After(50 * time.Millisecond):
			// expected
		}
	})

	t.Run("carriage return stripped", func(t *testing.T) {
		sink := NewLogSink(nil)
		sink.Write([]byte("hello\r\n"))

		select {
		case line := <-sink.Channel():
			if line != "hello" {
				t.Errorf("line = %q; want \"hello\"", line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}
	})

	t.Run("writes through to writer", func(t *testing.T) {
		w := &captureWriter{}
		sink := NewLogSink(w)
		sink.Write([]byte("test output\n"))

		// Drain channel
		select {
		case <-sink.Channel():
		case <-time.After(100 * time.Millisecond):
		}

		if !strings.Contains(w.data, "test output") {
			t.Errorf("writer got %q; want to contain \"test output\"", w.data)
		}
	})
}

// ---------------------------------------------------------------------------
// LogSink.Channel
// ---------------------------------------------------------------------------

func TestLogSinkChannel(t *testing.T) {
	sink := NewLogSink(nil)
	ch := sink.Channel()
	if ch == nil {
		t.Fatal("Channel() returned nil")
	}
	// Writing should populate the channel.
	sink.Write([]byte("ping\n"))
	select {
	case line := <-ch:
		if line != "ping" {
			t.Errorf("line = %q; want \"ping\"", line)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type captureWriter struct {
	data string
}

func (w *captureWriter) Write(p []byte) (int, error) {
	w.data += string(p)
	return len(p), nil
}
