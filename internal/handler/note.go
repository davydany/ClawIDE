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
	folderID := r.URL.Query().Get("folder_id")

	var notes []model.Note

	// Route to project store when projectID is provided
	if projectID != "" {
		ps, err := h.getProjectNoteStore(projectID)
		if err != nil {
			log.Printf("note list project store error: %v", err)
			// fallback to global store
			notes = h.noteStore.GetByProject(projectID)
		} else if q != "" {
			notes = ps.Search(q)
		} else if folderID != "" || r.URL.Query().Has("folder_id") {
			notes = ps.GetByFolder(folderID)
		} else {
			notes = ps.GetAll()
		}
	} else if q != "" {
		notes = h.noteStore.Search("", q)
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
		FolderID  string `json:"folder_id"`
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
	if err := model.ValidateNoteTitle(body.Title); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	n := model.Note{
		ID:        uuid.New().String(),
		ProjectID: body.ProjectID,
		FolderID:  body.FolderID,
		Title:     body.Title,
		Content:   body.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if body.ProjectID != "" {
		ps, err := h.getProjectNoteStore(body.ProjectID)
		if err != nil {
			log.Printf("note create project store error: %v", err)
			http.Error(w, "failed to access project store", http.StatusInternalServerError)
			return
		}
		if err := ps.Add(n); err != nil {
			log.Printf("note create error: %v", err)
			http.Error(w, "failed to create note", http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.noteStore.Add(n); err != nil {
			log.Printf("note create error: %v", err)
			http.Error(w, "failed to create note", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(n)
}

func (h *Handlers) UpdateNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "noteID")

	var body struct {
		ProjectID string `json:"project_id"`
		FolderID  *string `json:"folder_id"`
		Title     string `json:"title"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.ProjectID != "" {
		ps, err := h.getProjectNoteStore(body.ProjectID)
		if err != nil {
			http.Error(w, "failed to access project store", http.StatusInternalServerError)
			return
		}
		existing, ok := ps.Get(id)
		if !ok {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		if body.Title != "" {
			if err := model.ValidateNoteTitle(body.Title); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			existing.Title = body.Title
		}
		existing.Content = body.Content
		if body.FolderID != nil {
			existing.FolderID = *body.FolderID
		}
		existing.UpdatedAt = time.Now()

		if err := ps.Update(existing); err != nil {
			log.Printf("note update error: %v", err)
			http.Error(w, "failed to update note", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existing)
		return
	}

	// Global store fallback
	existing, ok := h.noteStore.Get(id)
	if !ok {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	if body.Title != "" {
		if err := model.ValidateNoteTitle(body.Title); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	projectID := r.URL.Query().Get("project_id")

	if projectID != "" {
		ps, err := h.getProjectNoteStore(projectID)
		if err != nil {
			http.Error(w, "failed to access project store", http.StatusInternalServerError)
			return
		}
		if err := ps.Delete(id); err != nil {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.noteStore.Delete(id); err != nil {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --------------- Note Folder Endpoints ---------------

func (h *Handlers) ListNoteFolders(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectNoteStore(projectID)
	if err != nil {
		http.Error(w, "failed to access project store", http.StatusInternalServerError)
		return
	}

	folders := ps.GetFolders()
	if folders == nil {
		folders = []model.Folder{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func (h *Handlers) CreateNoteFolder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		ParentID  string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := model.ValidateFolderName(body.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectNoteStore(body.ProjectID)
	if err != nil {
		http.Error(w, "failed to access project store", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	f := model.Folder{
		ID:        uuid.New().String(),
		Name:      body.Name,
		ParentID:  body.ParentID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := ps.CreateFolder(f); err != nil {
		log.Printf("note folder create error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(f)
}

func (h *Handlers) UpdateNoteFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "folderID")
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectNoteStore(projectID)
	if err != nil {
		http.Error(w, "failed to access project store", http.StatusInternalServerError)
		return
	}

	existing, ok := ps.GetFolder(id)
	if !ok {
		http.Error(w, "folder not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parent_id"`
		Order    *int    `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Name != "" {
		if err := model.ValidateFolderName(body.Name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing.Name = body.Name
	}
	if body.ParentID != nil {
		existing.ParentID = *body.ParentID
	}
	if body.Order != nil {
		existing.Order = *body.Order
	}
	existing.UpdatedAt = time.Now()

	if err := ps.UpdateFolder(existing); err != nil {
		log.Printf("note folder update error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeleteNoteFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "folderID")
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectNoteStore(projectID)
	if err != nil {
		http.Error(w, "failed to access project store", http.StatusInternalServerError)
		return
	}

	if err := ps.DeleteFolder(id); err != nil {
		http.Error(w, "folder not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ReorderNotes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string   `json:"project_id"`
		NoteIDs   []string `json:"note_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectNoteStore(body.ProjectID)
	if err != nil {
		http.Error(w, "failed to access project store", http.StatusInternalServerError)
		return
	}

	if err := ps.Reorder(body.NoteIDs); err != nil {
		log.Printf("note reorder error: %v", err)
		http.Error(w, "failed to reorder notes", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
