package handler

import (
	"encoding/json"
	"net/http"

	"github.com/davydany/ClawIDE/internal/version"
)

// Version returns the current build version as JSON.
func (h *Handlers) Version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version.Get())
}
