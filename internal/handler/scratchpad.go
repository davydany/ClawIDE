package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

func (h *Handlers) GetScratchpad(w http.ResponseWriter, r *http.Request) {
	pad := h.scratchpadStore.Get()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pad); err != nil {
		log.Printf("scratchpad get JSON encode error: %v", err)
	}
}

func (h *Handlers) UpdateScratchpad(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.scratchpadStore.Update(body.Content); err != nil {
		log.Printf("scratchpad update error: %v", err)
		http.Error(w, "failed to update scratchpad", http.StatusInternalServerError)
		return
	}

	pad := h.scratchpadStore.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pad)
}
