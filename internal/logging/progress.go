package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

// Progress provides user-visible feedback during long operations.
// The CLI implementation renders an animated spinner; the TUI already
// has its own progress via Bubble Tea, so it passes nil.
type Progress interface {
	// Start begins a spinner with an initial message.
	Start(msg string)
	// Update replaces the spinner message (e.g. "2/5 pages done").
	Update(msg string)
	// Stop removes the spinner line.
	Stop()
}

// IsTTY returns true when w is connected to an interactive terminal.
func IsTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

// ---------------------------------------------------------------------------
// CLIProgress — simple animated spinner for non-TUI CLI output
// ---------------------------------------------------------------------------

// spinner frames using Unicode Braille characters (compact, low-flicker).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// CLIProgress writes a single-line spinner to a terminal writer.
// It clears its own line before each write so slog output (which always
// ends with '\n') naturally pushes the spinner down.
type CLIProgress struct {
	w       io.Writer
	mu      sync.Mutex
	msg     string
	frame   int
	done    chan struct{}
	running bool
}

// NewCLIProgress creates a progress spinner that writes to w.
// Caller must ensure w is a terminal (use IsTTY first).
func NewCLIProgress(w io.Writer) *CLIProgress {
	return &CLIProgress{w: w}
}

func (p *CLIProgress) Start(msg string) {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		p.Update(msg)
		return
	}
	p.msg = msg
	p.frame = 0
	p.done = make(chan struct{})
	p.running = true
	p.mu.Unlock()

	go p.loop()
}

func (p *CLIProgress) Update(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.msg = msg
}

func (p *CLIProgress) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	close(p.done)
	p.mu.Unlock()

	// Clear the spinner line.
	fmt.Fprintf(p.w, "\r%s\r", strings.Repeat(" ", 80))
}

func (p *CLIProgress) loop() {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.mu.Lock()
			frame := spinnerFrames[p.frame%len(spinnerFrames)]
			msg := p.msg
			p.frame++
			p.mu.Unlock()
			// \r moves cursor to column 0; \033[K clears to end of line.
			fmt.Fprintf(p.w, "\r\033[K  %s %s", frame, msg)
		}
	}
}

// ProgressWriter wraps an io.Writer and coordinates with a CLIProgress
// so that slog output temporarily clears the spinner line, writes the
// log line, then redraws the spinner.
type ProgressWriter struct {
	inner    io.Writer
	progress *CLIProgress
}

// NewProgressWriter returns a writer that clears/redraws the spinner
// around every Write.  If progress is nil it passes through directly.
func NewProgressWriter(inner io.Writer, progress *CLIProgress) io.Writer {
	if progress == nil {
		return inner
	}
	return &ProgressWriter{inner: inner, progress: progress}
}

func (pw *ProgressWriter) Write(b []byte) (int, error) {
	pw.progress.mu.Lock()
	running := pw.progress.running
	if running {
		// Clear spinner line before log output.
		fmt.Fprintf(pw.inner, "\r\033[K")
	}
	n, err := pw.inner.Write(b)
	pw.progress.mu.Unlock()
	return n, err
}
