package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListVoiceBoxEntries(w http.ResponseWriter, r *http.Request) {
	entries := h.voiceBoxStore.GetAll()
	if entries == nil {
		entries = []model.VoiceBoxEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Printf("voicebox list JSON encode error: %v", err)
	}
}

func (h *Handlers) CreateVoiceBoxEntry(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	entry := model.VoiceBoxEntry{
		ID:        uuid.New().String(),
		Content:   body.Content,
		CreatedAt: time.Now(),
	}

	if err := h.voiceBoxStore.Add(entry); err != nil {
		log.Printf("voicebox create error: %v", err)
		http.Error(w, "failed to create voicebox entry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func (h *Handlers) DeleteVoiceBoxEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "entryID")

	if err := h.voiceBoxStore.Delete(id); err != nil {
		http.Error(w, "voicebox entry not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) DeleteAllVoiceBoxEntries(w http.ResponseWriter, r *http.Request) {
	if err := h.voiceBoxStore.DeleteAll(); err != nil {
		log.Printf("voicebox delete all error: %v", err)
		http.Error(w, "failed to clear voicebox history", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
