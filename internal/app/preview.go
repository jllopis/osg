package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"osg/internal/build"
	"osg/internal/config"
	"osg/internal/logging"
)

// PreviewOptions holds options for the preview command.
type PreviewOptions struct {
	FilePath string
	Port     int
	Timeout  time.Duration // inactivity timeout before auto-close
}

// RunPreview renders a single markdown file and serves it in the browser.
// The server auto-closes after the inactivity timeout.
func RunPreview(ctx context.Context, opts CLIOptions, previewOpts PreviewOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}

	// Resolve absolute path to the content file.
	absFile, err := filepath.Abs(previewOpts.FilePath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	absContent, err := filepath.Abs(cfg.ContentDir)
	if err != nil {
		return fmt.Errorf("resolve content dir: %w", err)
	}

	// The file must be inside the content directory.
	rel, err := filepath.Rel(absContent, absFile)
	if err != nil || len(rel) > 1 && rel[:2] == ".." {
		return fmt.Errorf("file %s is not inside content directory %s", absFile, absContent)
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)

	// Build the single page into a temp directory.
	tmpDir, err := build.PreviewBuild(cfg, absFile, logger)
	if err != nil {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Pick a port.
	port := previewOpts.Port
	if port == 0 {
		port, err = freePort()
		if err != nil {
			return fmt.Errorf("find free port: %w", err)
		}
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	timeout := previewOpts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Track activity for auto-close.
	activity := make(chan struct{}, 1)
	handler := activityMiddleware(http.FileServer(http.Dir(tmpDir)), activity)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Determine the page URL from the rendered output.
	pageURL := fmt.Sprintf("http://%s/", addr)
	// Walk tmpDir to find the first index.html and derive the path.
	_ = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == "index.html" {
			rel, _ := filepath.Rel(tmpDir, filepath.Dir(path))
			if rel != "." {
				pageURL = fmt.Sprintf("http://%s/%s/", addr, filepath.ToSlash(rel))
			}
			return filepath.SkipAll
		}
		return nil
	})

	fmt.Printf("Preview: %s\n", pageURL)
	fmt.Printf("Auto-close after %s of inactivity. Press Ctrl+C to stop.\n", timeout)

	// Start server in background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Open browser.
	openBrowser(pageURL)

	// Wait for inactivity timeout or context cancellation.
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			return nil
		case err := <-errCh:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		case <-activity:
			timer.Reset(timeout)
		case <-timer.C:
			fmt.Println("No activity, shutting down preview server.")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
			return nil
		}
	}
}

// activityMiddleware sends a signal on each HTTP request to reset the
// inactivity timer.
func activityMiddleware(next http.Handler, activity chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case activity <- struct{}{}:
		default:
		}
		next.ServeHTTP(w, r)
	})
}

// freePort asks the OS for an available port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

// openBrowser opens a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}
