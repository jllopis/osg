package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsLoopbackHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		host string
		want bool
	}{
		{"127.0.0.1:1314", true},
		{"127.0.0.1", true},
		{"localhost:1314", true},
		{"LocalHost", true},
		{"[::1]:1314", true},
		{"::1", true},
		{"evil.com", false},
		{"evil.com:1314", false},
		{"10.0.0.5:1314", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isLoopbackHost(tc.host); got != tc.want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

// TestSecurityMiddleware verifies the DNS-rebinding (Host) and CSRF
// (Sec-Fetch-Site / Origin) protections on the wrapped handler.
func TestSecurityMiddleware(t *testing.T) {
	s := newTestServer(t, newTestRunner(t))
	h := s.Handler()

	do := func(method, host string, headers map[string]string) int {
		req := httptest.NewRequest(method, "/operations/build/run", nil)
		req.Host = host
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("rejects non-loopback Host", func(t *testing.T) {
		if code := do(http.MethodGet, "evil.com", nil); code != http.StatusForbidden {
			t.Fatalf("non-loopback GET: code = %d, want 403", code)
		}
	})

	t.Run("rejects cross-site POST via Sec-Fetch-Site", func(t *testing.T) {
		code := do(http.MethodPost, "127.0.0.1:1314", map[string]string{"Sec-Fetch-Site": "cross-site"})
		if code != http.StatusForbidden {
			t.Fatalf("cross-site POST: code = %d, want 403", code)
		}
	})

	t.Run("rejects cross-origin POST via Origin", func(t *testing.T) {
		code := do(http.MethodPost, "127.0.0.1:1314", map[string]string{"Origin": "http://evil.com"})
		if code != http.StatusForbidden {
			t.Fatalf("cross-origin POST: code = %d, want 403", code)
		}
	})

	t.Run("allows same-origin POST", func(t *testing.T) {
		code := do(http.MethodPost, "127.0.0.1:1314", map[string]string{
			"Sec-Fetch-Site": "same-origin",
			"Origin":         "http://127.0.0.1:1314",
		})
		if code == http.StatusForbidden {
			t.Fatalf("same-origin POST was blocked (code %d)", code)
		}
	})

	t.Run("allows non-browser POST (no Origin, no Sec-Fetch-Site)", func(t *testing.T) {
		code := do(http.MethodPost, "127.0.0.1:1314", nil)
		if code == http.StatusForbidden {
			t.Fatalf("curl-style POST was blocked (code %d)", code)
		}
	})
}
