package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/davydany/ccmux/internal/middleware"
	"github.com/davydany/ccmux/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	sessions := h.store.GetSessions(project.ID)

	data := map[string]any{
		"Project":  project,
		"Sessions": sessions,
	}

	if err := h.renderer.RenderHTMX(w, r, "workspace", "session-list", data); err != nil {
		log.Printf("Error rendering sessions: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	branch := r.FormValue("branch")
	workDir := r.FormValue("work_dir")

	if name == "" {
		name = "Session " + time.Now().Format("15:04")
	}
	if workDir == "" {
		workDir = project.Path
	}

	sess := model.Session{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		Name:      name,
		Branch:    branch,
		WorkDir:   workDir,
		CreatedAt: time.Now(),
	}

	if err := h.store.AddSession(sess); err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		// Return the updated session list
		sessions := h.store.GetSessions(project.ID)
		data := map[string]any{
			"Project":  project,
			"Sessions": sessions,
		}
		if err := h.renderer.RenderHTMX(w, r, "workspace", "session-list", data); err != nil {
			log.Printf("Error rendering session list: %v", err)
		}
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/", http.StatusSeeOther)
}

func (h *Handlers) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sid")

	if err := h.store.DeleteSession(sessionID); err != nil {
		log.Printf("Error deleting session: %v", err)
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
