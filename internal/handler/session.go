package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
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

	paneID := uuid.New().String()
	now := time.Now()

	sess := model.Session{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		Name:      name,
		Branch:    branch,
		WorkDir:   workDir,
		Layout:    model.NewLeafPane(paneID),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.store.AddSession(sess); err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/projects/"+project.ID+"/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/", http.StatusSeeOther)
}

func (h *Handlers) RenameSession(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	sessionID := chi.URLParam(r, "sid")

	sess, ok := h.store.GetSession(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	sess.Name = name
	sess.UpdatedAt = time.Now()

	if err := h.store.UpdateSession(sess); err != nil {
		log.Printf("Error renaming session: %v", err)
		http.Error(w, "Failed to rename session", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/projects/"+project.ID+"/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/", http.StatusSeeOther)
}

func (h *Handlers) DeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sid")

	// Get session to find all panes before deleting
	sess, ok := h.store.GetSession(sessionID)
	if ok && sess.Layout != nil {
		// Destroy all pane tmux sessions
		for _, paneID := range sess.Layout.CollectLeaves() {
			if err := h.ptyManager.DestroySession(paneID); err != nil {
				log.Printf("Error destroying pane %s: %v", paneID, err)
			}
		}
	}

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
