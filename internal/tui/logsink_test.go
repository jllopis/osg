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
		sink := NewLogSink("general", nil)
		if sink == nil {
			t.Fatal("NewLogSink returned nil")
		}
		ch := sink.Channel()
		if ch == nil {
			t.Fatal("Channel() returned nil")
		}
		if sink.Source() != "general" {
			t.Errorf("Source() = %q; want \"general\"", sink.Source())
		}
	})

	t.Run("creates with writer", func(t *testing.T) {
		w := &captureWriter{}
		sink := NewLogSink("serve", w)
		if sink == nil {
			t.Fatal("NewLogSink returned nil")
		}
		if sink.Source() != "serve" {
			t.Errorf("Source() = %q; want \"serve\"", sink.Source())
		}
	})
}

// ---------------------------------------------------------------------------
// LogSink.Write
// ---------------------------------------------------------------------------

func TestLogSinkWrite(t *testing.T) {
	t.Run("complete line", func(t *testing.T) {
		sink := NewLogSink("test", nil)
		n, err := sink.Write([]byte("hello world\n"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 12 {
			t.Errorf("n = %d; want 12", n)
		}

		select {
		case tl := <-sink.Channel():
			if tl.Line != "hello world" {
				t.Errorf("line = %q; want \"hello world\"", tl.Line)
			}
			if tl.Source != "test" {
				t.Errorf("source = %q; want \"test\"", tl.Source)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}
	})

	t.Run("partial writes buffered", func(t *testing.T) {
		sink := NewLogSink("test", nil)
		_, _ = sink.Write([]byte("hel"))
		_, _ = sink.Write([]byte("lo\n"))

		select {
		case tl := <-sink.Channel():
			if tl.Line != "hello" {
				t.Errorf("line = %q; want \"hello\"", tl.Line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}
	})

	t.Run("multiple lines in one write", func(t *testing.T) {
		sink := NewLogSink("test", nil)
		_, _ = sink.Write([]byte("line1\nline2\n"))

		select {
		case tl := <-sink.Channel():
			if tl.Line != "line1" {
				t.Errorf("line1 = %q", tl.Line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out for line1")
		}

		select {
		case tl := <-sink.Channel():
			if tl.Line != "line2" {
				t.Errorf("line2 = %q", tl.Line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out for line2")
		}
	})

	t.Run("blank lines skipped", func(t *testing.T) {
		sink := NewLogSink("test", nil)
		_, _ = sink.Write([]byte("\n\nhello\n\n"))

		select {
		case tl := <-sink.Channel():
			if tl.Line != "hello" {
				t.Errorf("line = %q; want \"hello\"", tl.Line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}

		// Ensure no more lines.
		select {
		case tl := <-sink.Channel():
			t.Errorf("unexpected extra line: %q", tl.Line)
		case <-time.After(50 * time.Millisecond):
			// expected
		}
	})

	t.Run("carriage return stripped", func(t *testing.T) {
		sink := NewLogSink("test", nil)
		_, _ = sink.Write([]byte("hello\r\n"))

		select {
		case tl := <-sink.Channel():
			if tl.Line != "hello" {
				t.Errorf("line = %q; want \"hello\"", tl.Line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out waiting for line")
		}
	})

	t.Run("writes through to writer", func(t *testing.T) {
		w := &captureWriter{}
		sink := NewLogSink("test", w)
		_, _ = sink.Write([]byte("test output\n"))

		// Drain channel
		select {
		case <-sink.Channel():
		case <-time.After(100 * time.Millisecond):
		}

		if !strings.Contains(w.data, "test output") {
			t.Errorf("writer got %q; want to contain \"test output\"", w.data)
		}
	})

	t.Run("source tag preserved in channel", func(t *testing.T) {
		sink := NewLogSink("api", nil)
		_, _ = sink.Write([]byte("api request\n"))

		select {
		case tl := <-sink.Channel():
			if tl.Source != "api" {
				t.Errorf("Source = %q; want \"api\"", tl.Source)
			}
			if tl.Line != "api request" {
				t.Errorf("Line = %q; want \"api request\"", tl.Line)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timed out")
		}
	})
}

// ---------------------------------------------------------------------------
// LogSink.Channel
// ---------------------------------------------------------------------------

func TestLogSinkChannel(t *testing.T) {
	sink := NewLogSink("general", nil)
	ch := sink.Channel()
	if ch == nil {
		t.Fatal("Channel() returned nil")
	}
	// Writing should populate the channel.
	_, _ = sink.Write([]byte("ping\n"))
	select {
	case tl := <-ch:
		if tl.Line != "ping" {
			t.Errorf("line = %q; want \"ping\"", tl.Line)
		}
		if tl.Source != "general" {
			t.Errorf("source = %q; want \"general\"", tl.Source)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out")
	}
}

// ---------------------------------------------------------------------------
// MergeChannels
// ---------------------------------------------------------------------------

func TestMergeChannels(t *testing.T) {
	t.Run("merges multiple sinks", func(t *testing.T) {
		s1 := NewLogSink("serve", nil)
		s2 := NewLogSink("api", nil)
		merged := MergeChannels(s1, s2)

		_, _ = s1.Write([]byte("from serve\n"))
		_, _ = s2.Write([]byte("from api\n"))

		received := map[string]string{}
		for i := 0; i < 2; i++ {
			select {
			case tl := <-merged:
				received[tl.Source] = tl.Line
			case <-time.After(200 * time.Millisecond):
				t.Fatal("timed out waiting for merged line")
			}
		}

		if received["serve"] != "from serve" {
			t.Errorf("serve line = %q; want \"from serve\"", received["serve"])
		}
		if received["api"] != "from api" {
			t.Errorf("api line = %q; want \"from api\"", received["api"])
		}
	})

	t.Run("handles nil sinks", func(t *testing.T) {
		s1 := NewLogSink("serve", nil)
		merged := MergeChannels(nil, s1, nil)

		_, _ = s1.Write([]byte("ok\n"))

		select {
		case tl := <-merged:
			if tl.Line != "ok" {
				t.Errorf("line = %q; want \"ok\"", tl.Line)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatal("timed out")
		}
	})
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
