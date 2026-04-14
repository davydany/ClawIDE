package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/color"
	"github.com/davydany/ClawIDE/internal/docker"
	"github.com/davydany/ClawIDE/internal/editor"
	"github.com/davydany/ClawIDE/internal/fsutil"
	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// splitAndSortProjects splits projects into starred and non-starred groups,
// each sorted by SortOrder.
func splitAndSortProjects(projects []model.Project) (starred, nonStarred []model.Project) {
	for _, p := range projects {
		if p.Starred {
			starred = append(starred, p)
		} else {
			nonStarred = append(nonStarred, p)
		}
	}
	sort.Slice(starred, func(i, j int) bool {
		return starred[i].SortOrder < starred[j].SortOrder
	})
	sort.Slice(nonStarred, func(i, j int) bool {
		return nonStarred[i].SortOrder < nonStarred[j].SortOrder
	})
	return
}

func (h *Handlers) ListProjects(w http.ResponseWriter, r *http.Request) {
	starredProjects, unstarredProjects := splitAndSortProjects(h.store.GetProjects())
	data := map[string]any{
		"Title":           "ClawIDE - Projects",
		"Theme":           h.cfg.Theme,
		"Mode":            h.cfg.Mode,
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

	// Regenerate feature shades from the new project color.
	features := h.store.GetFeatures(project.ID)
	if len(features) > 0 {
		if body.Color == "" {
			// Clear all feature colors when project color is cleared.
			for _, f := range features {
				if f.Color != "" {
					f.Color = ""
					f.UpdatedAt = time.Now()
					if err := h.store.UpdateFeature(f); err != nil {
						log.Printf("Error clearing feature color %s: %v", f.ID, err)
					}
				}
			}
		} else {
			// Assign fresh shades to each feature sequentially.
			var usedColors []string
			for _, f := range features {
				shade, err := color.PickUnusedShade(body.Color, usedColors, 8)
				if err != nil {
					log.Printf("Error generating shade for feature %s: %v", f.ID, err)
					continue
				}
				f.Color = shade
				f.UpdatedAt = time.Now()
				if err := h.store.UpdateFeature(f); err != nil {
					log.Printf("Error updating feature shade %s: %v", f.ID, err)
				}
				usedColors = append(usedColors, shade)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (h *Handlers) ProjectWorkspace(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	sessions := h.store.GetSessions(project.ID)

	// Auto-create a default session when opening a project with no sessions
	if len(sessions) == 0 {
		now := time.Now()
		paneID := uuid.New().String()
		sess := model.Session{
			ID:        uuid.New().String(),
			ProjectID: project.ID,
			Name:      "Session " + now.Format("15:04"),
			WorkDir:   project.Path,
			Layout:    model.NewAgentPane(paneID),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := h.store.AddSession(sess); err != nil {
			log.Printf("Error auto-creating session: %v", err)
		} else {
			sessions = h.store.GetSessions(project.ID)
		}
	}

	features := h.store.GetFeatures(project.ID)

	isGitRepo := git.IsGitRepo(project.Path)
	var currentBranch string
	if isGitRepo {
		currentBranch, _ = git.CurrentBranch(project.Path)
	}

	// Collect starred and non-starred projects for quick-switch panel
	starredProjects, nonStarredProjects := splitAndSortProjects(h.store.GetProjects())

	// Build bar bookmark views for tab bar (project-scoped or global fallback)
	var barBookmarks []model.Bookmark
	if ps, err := h.getProjectBookmarkStore(project.ID); err == nil {
		barBookmarks = ps.GetRootBookmarks()
	} else {
		barBookmarks = h.bookmarkStore.GetRootByProject(project.ID)
	}
	var barBookmarkViews []BookmarkBarView
	for _, bm := range barBookmarks {
		barBookmarkViews = append(barBookmarkViews, BookmarkBarView{
			ID:         bm.ID,
			Name:       bm.Name,
			URL:        bm.URL,
			Emoji:      bm.Emoji,
			FaviconURL: bookmarkFaviconURL(bm.URL),
		})
	}

	data := map[string]any{
		"Title":              project.Name + " - ClawIDE",
		"Theme":           h.cfg.Theme,
		"Mode":            h.cfg.Mode,
		"Project":            project,
		"Sessions":           sessions,
		"Features":           features,
		"ActiveTab":          "terminal",
		"IsGitRepo":          isGitRepo,
		"CurrentBranch":      currentBranch,
		"StarredProjects":    starredProjects,
		"NonStarredProjects": nonStarredProjects,
		"BarBookmarks":       barBookmarkViews,
		"WebAppURL":          docker.FindWebAppURL(project.Path),
		"StartTour":            !h.cfg.WorkspaceTourCompleted,
		"ActiveFeatureID":      "",
		"ActiveBranch":         project.ActiveBranch,
		"SidebarPosition":      h.cfg.SidebarPosition,
		"SidebarWidth":         h.cfg.SidebarWidth,
		"PreferredEditor":      h.cfg.PreferredEditor,
		"PreferredEditorName":  editor.GetEditorName(h.cfg.PreferredEditor),
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

func (h *Handlers) ReorderProjects(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(body.IDs) == 0 {
		http.Error(w, "ids required", http.StatusBadRequest)
		return
	}
	if err := h.store.ReorderProjects(body.IDs); err != nil {
		log.Printf("Error reordering projects: %v", err)
		http.Error(w, "Failed to reorder projects", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "projectStarred")
	w.WriteHeader(http.StatusOK)
}

// RemoveProjectFromClawIDE clears a project from ClawIDE's state (cascading
// to sessions, features, and trashed features) but does NOT touch the
// project directory on disk. This is the "Remove from ClawIDE" menu action.
// DELETE /projects/{id}/
func (h *Handlers) RemoveProjectFromClawIDE(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	// Destroy any running PTYs before the store forgets about them.
	h.destroyProjectPTYs(projectID)

	if err := h.store.DeleteProject(projectID); err != nil {
		log.Printf("Error removing project from ClawIDE: %v", err)
		http.Error(w, "Failed to remove project", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RenameProject updates the display name of a project. Does not touch the
// filesystem. PATCH /projects/{id}/
func (h *Handlers) RenameProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	project.Name = name
	project.UpdatedAt = time.Now()

	if err := h.store.UpdateProject(project); err != nil {
		log.Printf("Error renaming project: %v", err)
		http.Error(w, "failed to rename project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// RenameProjectDirectory renames the project directory on disk within its
// current parent directory, updates project.Path, and rewrites Session.WorkDir
// for every session belonging to this project. It refuses to proceed if the
// project has any features, because feature worktree/clone paths are derived
// from the project path and renaming would orphan them.
// PATCH /projects/{id}/path
func (h *Handlers) RenameProjectDirectory(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	newBase := strings.TrimSpace(body.Name)
	if err := validateDirName(newBase); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Refuse if any features reference this project — their worktree/clone
	// paths are derived from project.Path and a rename would orphan them.
	if features := h.store.GetFeatures(projectID); len(features) > 0 {
		http.Error(w, "cannot rename directory: project has features/worktrees that depend on this path. Delete or merge them first.", http.StatusConflict)
		return
	}

	parent := filepath.Dir(project.Path)
	newPath := filepath.Join(parent, newBase)
	if newPath == project.Path {
		// No-op, still return success.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
		return
	}
	if _, err := os.Stat(newPath); err == nil {
		http.Error(w, "a file or directory already exists at "+newPath, http.StatusConflict)
		return
	}

	// Kill any running PTY sessions bound to the project before renaming,
	// because tmux bakes the cwd at session-create time.
	h.destroyProjectPTYs(projectID)

	oldPath := project.Path
	if err := os.Rename(oldPath, newPath); err != nil {
		log.Printf("Error renaming project directory %s -> %s: %v", oldPath, newPath, err)
		http.Error(w, "failed to rename directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	project.Path = newPath
	project.UpdatedAt = time.Now()
	if err := h.store.UpdateProject(project); err != nil {
		log.Printf("Error updating project after directory rename: %v", err)
		http.Error(w, "directory renamed but failed to update state: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdateSessionsWorkDir(projectID, oldPath, newPath); err != nil {
		log.Printf("Error updating session workdirs after rename: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// TrashProject moves a project's directory into the ClawIDE trash folder
// under the data dir, records a TrashedProject entry, and clears the project
// from the active store (cascading to sessions, features, and trashed
// features). Restorable from the trash UI within 30 days.
// POST /projects/{id}/trash
func (h *Handlers) TrashProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	project, ok := h.store.GetProject(projectID)
	if !ok {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	// Destroy any running PTYs before moving the directory out from under them.
	h.destroyProjectPTYs(projectID)

	trashID := uuid.New().String()
	trashRoot := filepath.Join(h.cfg.DataDir, "trash", "projects", trashID)
	trashPath := filepath.Join(trashRoot, filepath.Base(project.Path))

	if err := fsutil.MoveDir(project.Path, trashPath); err != nil {
		log.Printf("Error moving project %s to trash: %v", project.Path, err)
		http.Error(w, "failed to move project to trash: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tp := model.TrashedProject{
		ID:           trashID,
		Project:      project,
		OriginalPath: project.Path,
		TrashedPath:  trashPath,
		TrashedAt:    time.Now(),
	}
	if err := h.store.AddTrashedProject(tp); err != nil {
		log.Printf("Error recording trashed project: %v", err)
		// Attempt to roll back the move so the user's files aren't stranded.
		if rollbackErr := fsutil.MoveDir(trashPath, project.Path); rollbackErr != nil {
			log.Printf("Error rolling back trash move: %v", rollbackErr)
		} else {
			_ = os.RemoveAll(trashRoot)
		}
		http.Error(w, "failed to trash project", http.StatusInternalServerError)
		return
	}

	// Clear the project from the active store. This cascades to sessions,
	// features, and trashed features for this project.
	if err := h.store.DeleteProject(projectID); err != nil {
		log.Printf("Error clearing project state after trash: %v", err)
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// destroyProjectPTYs closes every PTY pane belonging to the given project.
// Mirrors the pattern used by DeleteFeature: iterate the project's sessions
// (both project-level and feature-level) and destroy each pane leaf.
func (h *Handlers) destroyProjectPTYs(projectID string) {
	for _, sess := range h.store.GetAllSessions() {
		if sess.ProjectID != projectID || sess.Layout == nil {
			continue
		}
		for _, paneID := range sess.Layout.CollectLeaves() {
			if err := h.ptyManager.DestroySession(paneID); err != nil {
				log.Printf("Error destroying pane %s: %v", paneID, err)
			}
		}
	}
}

// validateDirName enforces that a proposed directory basename is safe to use
// (no path separators, no traversal, no null bytes, not empty, not "." or "..").
func validateDirName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("name %q is reserved", name)
	}
	for _, c := range name {
		if c == '/' || c == '\\' || c == 0 {
			return fmt.Errorf("name contains an invalid character")
		}
	}
	return nil
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
		name := e.Name()
		// Skip hidden files and directories, and directories ending with -worktrees
		if e.IsDir() && name[0] != '.' && !strings.HasSuffix(name, "-worktrees") {
			dirs = append(dirs, DirEntry{
				Name: name,
				Path: filepath.Join(h.cfg.ProjectsDir, name),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ScanResult{Dirs: dirs})
}
