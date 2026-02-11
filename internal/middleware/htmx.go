package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const htmxKey contextKey = "is_htmx"

func HTMXDetect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isHTMX := r.Header.Get("HX-Request") == "true"
		ctx := context.WithValue(r.Context(), htmxKey, isHTMX)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func IsHTMX(r *http.Request) bool {
	v, _ := r.Context().Value(htmxKey).(bool)
	return v
}
