package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestIsTTY_Buffer(t *testing.T) {
	var buf bytes.Buffer
	if IsTTY(&buf) {
		t.Error("bytes.Buffer should not be a TTY")
	}
}

func TestCLIProgress_StartStopClearsLine(t *testing.T) {
	var buf bytes.Buffer
	p := NewCLIProgress(&buf)
	p.Start("working…")
	// Let the spinner tick at least once.
	time.Sleep(150 * time.Millisecond)
	p.Stop()

	out := buf.String()
	// The spinner should have written at least one frame.
	if len(out) == 0 {
		t.Fatal("expected spinner output, got empty string")
	}
	// After Stop, the spinner clears with spaces.
	if !strings.Contains(out, "working") {
		t.Error("expected spinner message in output")
	}
}

func TestCLIProgress_Update(t *testing.T) {
	var buf bytes.Buffer
	p := NewCLIProgress(&buf)
	p.Start("step 1")
	time.Sleep(100 * time.Millisecond)
	p.Update("step 2")
	time.Sleep(100 * time.Millisecond)
	p.Stop()

	out := buf.String()
	if !strings.Contains(out, "step 2") {
		t.Errorf("expected updated message 'step 2' in output:\n%s", out)
	}
}

func TestCLIProgress_DoubleStop(t *testing.T) {
	var buf bytes.Buffer
	p := NewCLIProgress(&buf)
	p.Start("test")
	time.Sleep(100 * time.Millisecond)
	p.Stop()
	p.Stop() // should not panic
}

func TestCLIProgress_StartWhileRunning(t *testing.T) {
	var buf bytes.Buffer
	p := NewCLIProgress(&buf)
	p.Start("first")
	p.Start("second") // should update, not start another goroutine
	time.Sleep(100 * time.Millisecond)
	p.Stop()

	out := buf.String()
	if !strings.Contains(out, "second") {
		t.Errorf("expected second message after re-Start, got:\n%s", out)
	}
}

func TestProgressWriter_PassesThrough(t *testing.T) {
	var buf bytes.Buffer
	// nil progress => pass-through.
	w := NewProgressWriter(&buf, nil)
	_, _ = w.Write([]byte("hello\n"))
	if buf.String() != "hello\n" {
		t.Errorf("expected pass-through, got %q", buf.String())
	}
}

func TestProgressWriter_ClearsSpinner(t *testing.T) {
	var buf bytes.Buffer
	p := NewCLIProgress(&buf)
	w := NewProgressWriter(&buf, p)

	p.Start("working")
	time.Sleep(100 * time.Millisecond)

	// A log write should clear the spinner line first.
	_, _ = w.Write([]byte("log line\n"))
	time.Sleep(100 * time.Millisecond)

	p.Stop()

	out := buf.String()
	if !strings.Contains(out, "log line") {
		t.Errorf("expected log output in buffer, got:\n%s", out)
	}
}
