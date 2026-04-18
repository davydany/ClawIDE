package middleware

import (
	"net/http"
	"strings"
)

// NoCacheAPI sets Cache-Control: no-store on all requests whose path contains
// "/api/". This prevents browsers from serving stale JSON responses from cache
// after mutations (POST/PUT/DELETE).
func NoCacheAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
