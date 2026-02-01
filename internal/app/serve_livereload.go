package app

import (
	"bytes"
	"net/http"
	"path"
	"strings"
	"sync"
)

const reloadEndpoint = "/__osg_reload"
const reloadScriptPath = "/__osg_reload.js"

type reloadHub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func newReloadHub() *reloadHub {
	return &reloadHub{clients: make(map[chan struct{}]struct{})}
}

func (h *reloadHub) Broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (h *reloadHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
	}()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case <-ch:
			_, _ = w.Write([]byte("data: reload\n\n"))
			flusher.Flush()
		}
	}
}

type liveReloadHandler struct {
	root    string
	base    http.Handler
	reloads *reloadHub
}

func newLiveReloadHandler(root string, base http.Handler, hub *reloadHub) http.Handler {
	return &liveReloadHandler{
		root:    root,
		base:    base,
		reloads: hub,
	}
}

func (h *liveReloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case reloadEndpoint:
		h.reloads.ServeHTTP(w, r)
		return
	case reloadScriptPath:
		serveReloadScript(w)
		return
	}

	rec := newResponseBuffer()
	h.base.ServeHTTP(rec, r)
	if rec.status != 0 && rec.status != http.StatusOK {
		rec.WriteTo(w)
		return
	}

	contentType := rec.header.Get("Content-Type")
	if !isHTMLResponse(r.URL.Path, contentType) {
		rec.WriteTo(w)
		return
	}

	body := injectReloadScript(rec.body.String())
	copyHeaders(w.Header(), rec.header)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func serveReloadScript(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`(() => {
  const source = new EventSource("` + reloadEndpoint + `");
  source.onmessage = () => window.location.reload();
})();`))
}

func injectReloadScript(html string) string {
	if strings.Contains(html, reloadScriptPath) {
		return html
	}
	script := `<script src="` + reloadScriptPath + `"></script>`
	if strings.Contains(html, "</body>") {
		return strings.Replace(html, "</body>", script+"</body>", 1)
	}
	return html + script
}

func isHTMLResponse(urlPath string, contentType string) bool {
	if strings.Contains(contentType, "text/html") {
		return true
	}
	clean := path.Clean(urlPath)
	if strings.HasSuffix(clean, ".html") || strings.HasSuffix(clean, ".htm") {
		return true
	}
	return strings.HasSuffix(clean, "/") || !strings.Contains(path.Base(clean), ".")
}

type responseBuffer struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{header: make(http.Header)}
}

func (r *responseBuffer) Header() http.Header {
	return r.header
}

func (r *responseBuffer) WriteHeader(status int) {
	r.status = status
}

func (r *responseBuffer) Write(p []byte) (int, error) {
	return r.body.Write(p)
}

func (r *responseBuffer) WriteTo(w http.ResponseWriter) {
	copyHeaders(w.Header(), r.header)
	if r.status != 0 {
		w.WriteHeader(r.status)
	}
	_, _ = w.Write(r.body.Bytes())
}

func copyHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
