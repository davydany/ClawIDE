package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListSnippets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var snippets []model.Snippet
	if q != "" {
		snippets = h.snippetStore.Search(q)
	} else {
		snippets = h.snippetStore.GetAll()
	}
	if snippets == nil {
		snippets = []model.Snippet{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(snippets); err != nil {
		log.Printf("snippet list JSON encode error: %v", err)
	}
}

func (h *Handlers) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	now := time.Now()
	sn := model.Snippet{
		ID:        uuid.New().String(),
		Name:      body.Name,
		Content:   body.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.snippetStore.Add(sn); err != nil {
		log.Printf("snippet create error: %v", err)
		http.Error(w, "failed to create snippet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sn)
}

func (h *Handlers) UpdateSnippet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "snippetID")

	existing, ok := h.snippetStore.Get(id)
	if !ok {
		http.Error(w, "snippet not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Name != "" {
		existing.Name = body.Name
	}
	existing.Content = body.Content
	existing.UpdatedAt = time.Now()

	if err := h.snippetStore.Update(existing); err != nil {
		log.Printf("snippet update error: %v", err)
		http.Error(w, "failed to update snippet", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeleteSnippet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "snippetID")

	if err := h.snippetStore.Delete(id); err != nil {
		http.Error(w, "snippet not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
