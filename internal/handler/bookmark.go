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

const maxInBarPerProject = 5

// StarredBookmarkView is the template-friendly representation of a bar bookmark.
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
	folderID := r.URL.Query().Get("folder_id")

	var bookmarks []model.Bookmark

	if projectID != "" {
		ps, err := h.getProjectBookmarkStore(projectID)
		if err != nil {
			log.Printf("bookmark list project store error: %v", err)
			bookmarks = h.bookmarkStore.GetByProject(projectID)
		} else if q != "" {
			bookmarks = ps.Search(q)
		} else if folderID != "" || r.URL.Query().Has("folder_id") {
			bookmarks = ps.GetByFolder(folderID)
		} else {
			bookmarks = ps.GetAll()
		}
	} else if q != "" && projectID != "" {
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
		FolderID  string `json:"folder_id"`
		Name      string `json:"name"`
		URL       string `json:"url"`
		Emoji     string `json:"emoji"`
		InBar     bool   `json:"in_bar"`
		Starred   bool   `json:"starred"` // backward compat
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

	// Accept either in_bar or starred
	inBar := body.InBar || body.Starred

	now := time.Now()
	b := model.Bookmark{
		ID:        uuid.New().String(),
		ProjectID: body.ProjectID,
		FolderID:  body.FolderID,
		Name:      body.Name,
		URL:       body.URL,
		Emoji:     body.Emoji,
		InBar:     inBar,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Try project store first
	ps, err := h.getProjectBookmarkStore(body.ProjectID)
	if err != nil {
		log.Printf("bookmark create project store error: %v", err)
		// fallback to global
		if inBar {
			count := h.bookmarkStore.CountStarredByProject(body.ProjectID)
			if count >= maxInBarPerProject {
				http.Error(w, "maximum of 5 bar bookmarks per project", http.StatusBadRequest)
				return
			}
		}
		b.Starred = inBar // keep backward compat for global store
		if err := h.bookmarkStore.Add(b); err != nil {
			log.Printf("bookmark create error: %v", err)
			http.Error(w, "failed to create bookmark", http.StatusInternalServerError)
			return
		}
	} else {
		if inBar {
			count := ps.CountInBar()
			if count >= maxInBarPerProject {
				http.Error(w, "maximum of 5 bar bookmarks per project", http.StatusBadRequest)
				return
			}
		}
		if err := ps.Add(b); err != nil {
			log.Printf("bookmark create error: %v", err)
			http.Error(w, "failed to create bookmark", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (h *Handlers) UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookmarkID")
	projectID := r.URL.Query().Get("project_id")

	var body struct {
		ProjectID string  `json:"project_id"`
		FolderID  *string `json:"folder_id"`
		Name      string  `json:"name"`
		URL       string  `json:"url"`
		Emoji     string  `json:"emoji"`
		InBar     *bool   `json:"in_bar"`
		Starred   *bool   `json:"starred"` // backward compat
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Use project_id from body or query param
	pid := body.ProjectID
	if pid == "" {
		pid = projectID
	}

	if pid != "" {
		ps, err := h.getProjectBookmarkStore(pid)
		if err == nil {
			existing, ok := ps.Get(id)
			if !ok {
				http.Error(w, "bookmark not found", http.StatusNotFound)
				return
			}
			if body.Name != "" {
				existing.Name = body.Name
			}
			if body.URL != "" {
				existing.URL = body.URL
			}
			existing.Emoji = body.Emoji
			if body.FolderID != nil {
				existing.FolderID = *body.FolderID
			}

			// Handle InBar or Starred toggle
			if body.InBar != nil && *body.InBar != existing.InBar {
				if *body.InBar {
					count := ps.CountInBar()
					if count >= maxInBarPerProject {
						http.Error(w, "maximum of 5 bar bookmarks per project", http.StatusBadRequest)
						return
					}
				}
				existing.InBar = *body.InBar
			} else if body.Starred != nil {
				newVal := *body.Starred
				if newVal != existing.InBar {
					if newVal {
						count := ps.CountInBar()
						if count >= maxInBarPerProject {
							http.Error(w, "maximum of 5 bar bookmarks per project", http.StatusBadRequest)
							return
						}
					}
					existing.InBar = newVal
				}
			}
			existing.UpdatedAt = time.Now()

			if err := ps.Update(existing); err != nil {
				log.Printf("bookmark update error: %v", err)
				http.Error(w, "failed to update bookmark", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(existing)
			return
		}
	}

	// Global store fallback
	existing, ok := h.bookmarkStore.Get(id)
	if !ok {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	if body.Name != "" {
		existing.Name = body.Name
	}
	if body.URL != "" {
		existing.URL = body.URL
	}
	existing.Emoji = body.Emoji

	if body.Starred != nil && *body.Starred != existing.Starred {
		if *body.Starred {
			count := h.bookmarkStore.CountStarredByProject(existing.ProjectID)
			if count >= maxInBarPerProject {
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
	projectID := r.URL.Query().Get("project_id")

	if projectID != "" {
		ps, err := h.getProjectBookmarkStore(projectID)
		if err == nil {
			if err := ps.Delete(id); err != nil {
				http.Error(w, "bookmark not found", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	if err := h.bookmarkStore.Delete(id); err != nil {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ToggleBookmarkStar(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "bookmarkID")
	projectID := r.URL.Query().Get("project_id")

	if projectID != "" {
		ps, err := h.getProjectBookmarkStore(projectID)
		if err == nil {
			existing, ok := ps.Get(id)
			if !ok {
				http.Error(w, "bookmark not found", http.StatusNotFound)
				return
			}
			if !existing.InBar {
				count := ps.CountInBar()
				if count >= maxInBarPerProject {
					http.Error(w, "maximum of 5 bar bookmarks per project", http.StatusBadRequest)
					return
				}
			}
			existing.InBar = !existing.InBar
			existing.UpdatedAt = time.Now()

			if err := ps.Update(existing); err != nil {
				log.Printf("bookmark bar toggle error: %v", err)
				http.Error(w, "failed to toggle bar", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(existing)
			return
		}
	}

	// Global fallback
	existing, ok := h.bookmarkStore.Get(id)
	if !ok {
		http.Error(w, "bookmark not found", http.StatusNotFound)
		return
	}
	if !existing.Starred {
		count := h.bookmarkStore.CountStarredByProject(existing.ProjectID)
		if count >= maxInBarPerProject {
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

// --------------- Bookmark Folder Endpoints ---------------

func (h *Handlers) ListBookmarkFolders(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectBookmarkStore(projectID)
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

func (h *Handlers) CreateBookmarkFolder(w http.ResponseWriter, r *http.Request) {
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
	if body.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectBookmarkStore(body.ProjectID)
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
		log.Printf("bookmark folder create error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(f)
}

func (h *Handlers) UpdateBookmarkFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "folderID")
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectBookmarkStore(projectID)
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
		log.Printf("bookmark folder update error: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

func (h *Handlers) DeleteBookmarkFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "folderID")
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectBookmarkStore(projectID)
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

func (h *Handlers) ReorderBookmarks(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID   string   `json:"project_id"`
		BookmarkIDs []string `json:"bookmark_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.ProjectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	ps, err := h.getProjectBookmarkStore(body.ProjectID)
	if err != nil {
		http.Error(w, "failed to access project store", http.StatusInternalServerError)
		return
	}

	if err := ps.Reorder(body.BookmarkIDs); err != nil {
		log.Printf("bookmark reorder error: %v", err)
		http.Error(w, "failed to reorder bookmarks", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
