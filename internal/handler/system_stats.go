package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/sysinfo"
)

// SystemStats returns system-level statistics as JSON.
func (h *Handlers) SystemStats(w http.ResponseWriter, r *http.Request) {
	stats := sysinfo.Gather(h.store)
	stats.ServerPort = h.cfg.Port

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("system stats JSON encode error: %v", err)
	}
}
