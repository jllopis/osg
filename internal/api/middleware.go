package api

import (
	"net/http"
	"strings"
)

// CORSMiddleware returns a handler that sets CORS headers and handles
// preflight OPTIONS requests. If allowedOrigins is empty, CORS headers
// are not set (same-origin only). When withCredentials is true (comments
// enabled), Access-Control-Allow-Credentials is set and wildcard origins
// are expanded to the request origin.
func CORSMiddleware(allowedOrigins []string, withCredentials bool, next http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		return next
	}

	originSet := make(map[string]bool, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	allowedMethods := "POST, OPTIONS"
	if withCredentials {
		allowedMethods = "GET, POST, DELETE, OPTIONS"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		allowed := allowAll || originSet[origin]
		if !allowed {
			next.ServeHTTP(w, r)
			return
		}

		// When credentials are needed, cannot use wildcard origin — must
		// echo the specific origin.
		if withCredentials || !allowAll {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if withCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
