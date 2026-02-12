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
	var starredProjects, unstarredProjects []model.Project
	for _, p := range projects {
		if p.Starred {
			starredProjects = append(starredProjects, p)
		} else {
			unstarredProjects = append(unstarredProjects, p)
		}
	}
	data := map[string]any{
		"Title":           "ClawIDE - Projects",
		"StarredProjects": starredProjects,
		"Projects":        unstarredProjects,
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

func (h *Handlers) UpdateProjectColor(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	var body struct {
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate hex color format if non-empty
	if body.Color != "" {
		if len(body.Color) != 7 || body.Color[0] != '#' {
			http.Error(w, "color must be a hex value like #ff0000 or empty to clear", http.StatusBadRequest)
			return
		}
		for _, c := range body.Color[1:] {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				http.Error(w, "color must be a valid hex value", http.StatusBadRequest)
				return
			}
		}
	}

	project.Color = body.Color
	project.UpdatedAt = time.Now()

	if err := h.store.UpdateProject(project); err != nil {
		log.Printf("Error updating project color: %v", err)
		http.Error(w, "failed to update color", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
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

	// Collect starred projects for quick-switch panel
	var starredProjects []model.Project
	for _, p := range h.store.GetProjects() {
		if p.Starred {
			starredProjects = append(starredProjects, p)
		}
	}

	// Build starred bookmark views for tab bar
	starredBookmarks := h.bookmarkStore.GetStarredByProject(project.ID)
	var starredBookmarkViews []StarredBookmarkView
	for _, bm := range starredBookmarks {
		starredBookmarkViews = append(starredBookmarkViews, StarredBookmarkView{
			ID:         bm.ID,
			Name:       bm.Name,
			URL:        bm.URL,
			Emoji:      bm.Emoji,
			FaviconURL: bookmarkFaviconURL(bm.URL),
		})
	}

	data := map[string]any{
		"Title":             project.Name + " - ClawIDE",
		"Project":           project,
		"Sessions":          sessions,
		"Features":          features,
		"ActiveTab":         "terminal",
		"IsGitRepo":         isGitRepo,
		"CurrentBranch":     currentBranch,
		"StarredProjects":   starredProjects,
		"StarredBookmarks":  starredBookmarkViews,
		"StartTour":         !h.cfg.WorkspaceTourCompleted,
		"ActiveFeatureID":   "",
		"SidebarPosition":   h.cfg.SidebarPosition,
		"SidebarWidth":      h.cfg.SidebarWidth,
	}

	if err := h.renderer.RenderHTMX(w, r, "workspace", "workspace", data); err != nil {
		log.Printf("Error rendering workspace: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *Handlers) ToggleStar(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	if _, err := h.store.ToggleProjectStar(projectID); err != nil {
		log.Printf("Error toggling star: %v", err)
		http.Error(w, "Failed to toggle star", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "projectStarred")

	project, _ := h.store.GetProject(projectID)

	if err := h.renderer.RenderHTMX(w, r, "project-list", "star-button", project); err != nil {
		log.Printf("Error rendering star button: %v", err)
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
