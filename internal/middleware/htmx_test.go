package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMXDetect(t *testing.T) {
	t.Run("with HX-Request header", func(t *testing.T) {
		var capturedHTMX bool
		handler := HTMXDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHTMX = IsHTMX(r)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.True(t, capturedHTMX)
	})

	t.Run("without HX-Request header", func(t *testing.T) {
		var capturedHTMX bool
		handler := HTMXDetect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHTMX = IsHTMX(r)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.False(t, capturedHTMX)
	})
}

func TestIsHTMX(t *testing.T) {
	t.Run("missing context value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		assert.False(t, IsHTMX(req))
	})
}
