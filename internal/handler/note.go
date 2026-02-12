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

func (h *Handlers) ListNotes(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	q := r.URL.Query().Get("q")

	var notes []model.Note
	if q != "" {
		notes = h.noteStore.Search(projectID, q)
	} else if projectID != "" {
		notes = h.noteStore.GetByProject(projectID)
	} else {
		notes = h.noteStore.GetAll()
	}
	if notes == nil {
		notes = []model.Note{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notes); err != nil {
		log.Printf("note list JSON encode error: %v", err)
	}
}

func (h *Handlers) CreateNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	now := time.Now()
	n := model.Note{
		ID:        uuid.New().String(),
		ProjectID: body.ProjectID,
		Title:     body.Title,
		Content:   body.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.noteStore.Add(n); err != nil {
		log.Printf("note create error: %v", err)
		http.Error(w, "failed to create note", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(n)
}

func (h *Handlers) UpdateNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")

	existing, ok := h.noteStore.Get(id)
	if !ok {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}

	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Title != "" {
		existing.Title = body.Title
	}
	existing.Content = body.Content
	existing.UpdatedAt = time.Now()

	if err := h.noteStore.Update(existing); err != nil {
		log.Printf("note update error: %v", err)
		http.Error(w, "failed to update note", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeleteNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")

	if err := h.noteStore.Delete(id); err != nil {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
