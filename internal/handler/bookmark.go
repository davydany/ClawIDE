package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxStarredPerProject = 5

// StarredBookmarkView is the template-friendly representation of a starred bookmark.
type StarredBookmarkView struct {
	ID         string
	Name       string
	URL        string
	Emoji      string
	FaviconURL string
}

func bookmarkFaviconURL(bookmarkURL string) string {
	parsed, err := url.Parse(bookmarkURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return "https://www.google.com/s2/favicons?domain=" + parsed.Host + "&sz=32"
}

func (h *Handlers) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	q := r.URL.Query().Get("q")

	var bookmarks []model.Bookmark
	if q != "" && projectID != "" {
		bookmarks = h.bookmarkStore.Search(projectID, q)
	} else if projectID != "" {
		bookmarks = h.bookmarkStore.GetByProject(projectID)
	} else {
		bookmarks = h.bookmarkStore.GetAll()
	}
	if bookmarks == nil {
		bookmarks = []model.Bookmark{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bookmarks); err != nil {
		log.Printf("bookmark list JSON encode error: %v", err)
	}
}

func (h *Handlers) CreateBookmark(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		Emoji     string `json:"emoji"`
		Starred   bool   `json:"starred"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if body.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if body.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	// Enforce max starred
	if body.Starred {
		count := h.bookmarkStore.CountStarredByProject(body.ProjectID)
		if count >= maxStarredPerProject {
			http.Error(w, "maximum of 5 starred bookmarks per project", http.StatusBadRequest)
			return
		}
	}

	now := time.Now()
	b := model.Bookmark{
		ID:        uuid.New().String(),
		ProjectID: body.ProjectID,
		Name:      body.Name,
		URL:       body.URL,
		Emoji:     body.Emoji,
		Starred:   body.Starred,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.bookmarkStore.Add(b); err != nil {
		log.Printf("bookmark create error: %v", err)
		http.Error(w, "failed to create bookmark", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (h *Handlers) UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookmarkID")

	existing, ok := h.bookmarkStore.Get(id)
	if !ok {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Emoji   string `json:"emoji"`
		Starred *bool  `json:"starred"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Name != "" {
		existing.Name = body.Name
	}
	if body.URL != "" {
		existing.URL = body.URL
	}
	// Emoji can be cleared explicitly
	existing.Emoji = body.Emoji

	if body.Starred != nil && *body.Starred != existing.Starred {
		if *body.Starred {
			count := h.bookmarkStore.CountStarredByProject(existing.ProjectID)
			if count >= maxStarredPerProject {
				http.Error(w, "maximum of 5 starred bookmarks per project", http.StatusBadRequest)
				return
			}
		}
		existing.Starred = *body.Starred
	}

	existing.UpdatedAt = time.Now()

	if err := h.bookmarkStore.Update(existing); err != nil {
		log.Printf("bookmark update error: %v", err)
		http.Error(w, "failed to update bookmark", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookmarkID")

	if err := h.bookmarkStore.Delete(id); err != nil {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ToggleBookmarkStar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookmarkID")

	existing, ok := h.bookmarkStore.Get(id)
	if !ok {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}

	if !existing.Starred {
		count := h.bookmarkStore.CountStarredByProject(existing.ProjectID)
		if count >= maxStarredPerProject {
			http.Error(w, "maximum of 5 starred bookmarks per project", http.StatusBadRequest)
			return
		}
	}

	existing.Starred = !existing.Starred
	existing.UpdatedAt = time.Now()

	if err := h.bookmarkStore.Update(existing); err != nil {
		log.Printf("bookmark star toggle error: %v", err)
		http.Error(w, "failed to toggle star", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}
