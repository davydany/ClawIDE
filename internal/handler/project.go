package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects := h.store.GetProjects()
	data := map[string]any{
		"Title":    "ClawIDE - Projects",
		"Projects": projects,
	}
	if err := h.renderer.RenderHTMX(w, r, "project-list", "project-list", data); err != nil {
		log.Printf("Error rendering projects: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	path := r.FormValue("path")

	if name == "" || path == "" {
		http.Error(w, "Name and path are required", http.StatusBadRequest)
		return
	}

	// Expand ~ in path
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	// Verify path exists
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		http.Error(w, "Path does not exist or is not a directory", http.StatusBadRequest)
		return
	}

	now := time.Now()
	project := model.Project{
		ID:        uuid.New().String(),
		Name:      name,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.store.AddProject(project); err != nil {
		log.Printf("Error creating project: %v", err)
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	// If htmx, return updated project list
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) ProjectWorkspace(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	sessions := h.store.GetSessions(project.ID)
	features := h.store.GetFeatures(project.ID)

	isGitRepo := git.IsGitRepo(project.Path)
	var currentBranch string
	if isGitRepo {
		currentBranch, _ = git.CurrentBranch(project.Path)
	}

	data := map[string]any{
		"Title":         project.Name + " - ClawIDE",
		"Project":       project,
		"Sessions":      sessions,
		"Features":      features,
		"ActiveTab":     "terminal",
		"IsGitRepo":     isGitRepo,
		"CurrentBranch": currentBranch,
		"StartTour":     !h.cfg.WorkspaceTourCompleted,
	}

	if err := h.renderer.RenderHTMX(w, r, "workspace", "workspace", data); err != nil {
		log.Printf("Error rendering workspace: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) DeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	if err := h.store.DeleteProject(projectID); err != nil {
		log.Printf("Error deleting project: %v", err)
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type ScanResult struct {
	Dirs []DirEntry `json:"dirs"`
}

type DirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (h *Handlers) ScanProjects(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(h.cfg.ProjectsDir)
	if err != nil {
		http.Error(w, "Failed to scan projects directory", http.StatusInternalServerError)
		return
	}

	var dirs []DirEntry
	for _, e := range entries {
		if e.IsDir() && e.Name()[0] != '.' {
			dirs = append(dirs, DirEntry{
				Name: e.Name(),
				Path: filepath.Join(h.cfg.ProjectsDir, e.Name()),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ScanResult{Dirs: dirs})
}
