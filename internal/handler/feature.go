package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/davydany/ClawIDE/internal/color"
	"github.com/davydany/ClawIDE/internal/editor"
	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CreateFeature creates a new feature or branch workspace with its own git
// branch and isolated working directory.
// POST /projects/{id}/features/
func (h *Handlers) CreateFeature(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)

	if !git.IsGitRepo(project.Path) {
		http.Error(w, "project path is not a git repository", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	baseBranch := r.FormValue("base_branch")
	featureType := r.FormValue("type")
	prefix := r.FormValue("prefix")

	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Default to feature type for backward compatibility.
	if featureType == "" {
		featureType = model.FeatureTypeFeature
	}

	// Determine the base branch: explicit > project active branch > current branch.
	if baseBranch == "" {
		if project.ActiveBranch != "" {
			baseBranch = project.ActiveBranch
		} else {
			current, err := git.CurrentBranch(project.Path)
			if err != nil || current == "" {
				http.Error(w, "could not determine current branch", http.StatusInternalServerError)
				return
			}
			baseBranch = current
		}
	}

	// Determine branch name and working directory based on type.
	var branchName string
	var workDir string

	switch featureType {
	case model.FeatureTypeBranch:
		branchName = git.SanitizeBranchNameWithPrefix(name, prefix)
		workDir = git.CloneDir(project.Path, branchName)
	default:
		featureType = model.FeatureTypeFeature
		branchName = git.SanitizeBranchName(name)
		workDir = git.WorktreeDir(project.Path, branchName)
	}

	// Create the git branch from base.
	if err := git.CreateBranch(project.Path, branchName, baseBranch); err != nil {
		log.Printf("Error creating branch %q: %v", branchName, err)
		http.Error(w, "failed to create branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Switch back to the base branch so the main worktree isn't on the
	// new branch.
	if err := git.CheckoutBranch(project.Path, baseBranch); err != nil {
		log.Printf("Error switching back to base branch %q: %v", baseBranch, err)
	}

	// Create the isolated working directory.
	switch featureType {
	case model.FeatureTypeBranch:
		if err := git.CloneLocal(project.Path, workDir, branchName); err != nil {
			log.Printf("Error creating clone: %v", err)
			http.Error(w, "failed to create clone: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		if err := git.CreateWorktree(project.Path, branchName, workDir); err != nil {
			log.Printf("Error creating worktree: %v", err)
			http.Error(w, "failed to create worktree: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	now := time.Now()
	featureID := uuid.New().String()

	feature := model.Feature{
		ID:           featureID,
		ProjectID:    project.ID,
		Type:         featureType,
		Name:         name,
		BranchName:   branchName,
		BaseBranch:   baseBranch,
		WorktreePath: workDir,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Auto-assign a shade of the project color if the project has one.
	if project.Color != "" {
		existingFeatures := h.store.GetFeatures(project.ID)
		var usedColors []string
		for _, ef := range existingFeatures {
			if ef.Color != "" {
				usedColors = append(usedColors, ef.Color)
			}
		}
		if shade, err := color.PickUnusedShade(project.Color, usedColors, 8); err == nil {
			feature.Color = shade
		}
	}

	if err := h.store.AddFeature(feature); err != nil {
		log.Printf("Error storing feature: %v", err)
		http.Error(w, "failed to store feature", http.StatusInternalServerError)
		return
	}

	// Create an initial session in the workspace.
	paneID := uuid.New().String()
	sess := model.Session{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		FeatureID: featureID,
		Name:      "Session " + time.Now().Format("15:04"),
		WorkDir:   workDir,
		Layout:    model.NewAgentPane(paneID),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.store.AddSession(sess); err != nil {
		log.Printf("Error creating initial session: %v", err)
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/features/"+featureID+"/", http.StatusSeeOther)
}

// UpdateFeatureColor updates the color of a feature.
// PATCH /projects/{id}/features/{fid}/color
func (h *Handlers) UpdateFeatureColor(w http.ResponseWriter, r *http.Request) {
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
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

	feature.Color = body.Color
	feature.UpdatedAt = time.Now()

	if err := h.store.UpdateFeature(feature); err != nil {
		log.Printf("Error updating feature color: %v", err)
		http.Error(w, "failed to update color", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feature)
}

// FeatureWorkspace renders the feature workspace page.
// GET /projects/{id}/features/{fid}/
func (h *Handlers) FeatureWorkspace(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	sessions := h.store.GetFeatureSessions(featureID)

	// Collect starred and non-starred projects for quick-switch panel
	starredProjects, nonStarredProjects := splitAndSortProjects(h.store.GetProjects())

	features := h.store.GetFeatures(project.ID)

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
		"Title":              feature.Name + " - " + project.Name + " - ClawIDE",
		"Theme":           h.cfg.Theme,
		"Mode":            h.cfg.Mode,
		"Project":            project,
		"Feature":            feature,
		"Features":           features,
		"Sessions":           sessions,
		"ActiveTab":          "terminal",
		"StarredProjects":    starredProjects,
		"NonStarredProjects": nonStarredProjects,
		"BarBookmarks":       barBookmarkViews,
		"ActiveFeatureID":    featureID,
		"ActiveBranch":         project.ActiveBranch,
		"IsGitRepo":            true,
		"SidebarPosition":      h.cfg.SidebarPosition,
		"SidebarWidth":         h.cfg.SidebarWidth,
		"AIReviewCommand":      h.cfg.AIReviewCommand,
		"PreferredEditor":      h.cfg.PreferredEditor,
		"PreferredEditorName":  editor.GetEditorName(h.cfg.PreferredEditor),
	}

	if err := h.renderer.RenderHTMX(w, r, "feature-workspace", "feature-workspace", data); err != nil {
		log.Printf("Error rendering feature workspace: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DeleteFeature soft-deletes a feature by moving it to the trash bin.
// The git branch is preserved so the feature can be restored later.
// Trashed features are automatically purged after 30 days.
// DELETE /projects/{id}/features/{fid}/
func (h *Handlers) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	// Destroy all PTY sessions belonging to this workspace.
	sessions := h.store.GetFeatureSessions(featureID)
	for _, sess := range sessions {
		if sess.Layout != nil {
			for _, paneID := range sess.Layout.CollectLeaves() {
				if err := h.ptyManager.DestroySession(paneID); err != nil {
					log.Printf("Error destroying pane %s: %v", paneID, err)
				}
			}
		}
	}

	// Remove the working directory but keep the branch for future restoration.
	if feature.IsClone() {
		if err := git.RemoveClone(feature.WorktreePath); err != nil {
			log.Printf("Error removing clone %s: %v", feature.WorktreePath, err)
		}
	} else {
		if err := git.RemoveWorktree(project.Path, feature.WorktreePath); err != nil {
			log.Printf("Error removing worktree %s: %v", feature.WorktreePath, err)
		}
	}

	// Move the feature to trash.
	tf := model.TrashedFeature{
		ID:          uuid.New().String(),
		Feature:     feature,
		ProjectID:   project.ID,
		ProjectName: project.Name,
		ProjectPath: project.Path,
		TrashedAt:   time.Now(),
	}
	if err := h.store.AddTrashedFeature(tf); err != nil {
		log.Printf("Error trashing feature: %v", err)
		http.Error(w, "failed to trash feature", http.StatusInternalServerError)
		return
	}

	// Delete the feature from the active store (cascades to sessions).
	if err := h.store.DeleteFeature(featureID); err != nil {
		log.Printf("Error deleting feature: %v", err)
		http.Error(w, "failed to delete feature", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/projects/"+project.ID+"/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/", http.StatusSeeOther)
}

// CreateFeatureSession creates a new session scoped to a feature workspace.
// POST /projects/{id}/features/{fid}/sessions/
func (h *Handlers) CreateFeatureSession(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = "Session " + time.Now().Format("15:04")
	}

	paneID := uuid.New().String()
	now := time.Now()

	sess := model.Session{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		FeatureID: featureID,
		Name:      name,
		WorkDir:   feature.WorktreePath,
		Layout:    model.NewAgentPane(paneID),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.store.AddSession(sess); err != nil {
		log.Printf("Error creating feature session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/features/"+featureID+"/", http.StatusSeeOther)
}
