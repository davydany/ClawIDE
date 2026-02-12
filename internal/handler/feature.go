package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CreateFeature creates a new feature workspace with its own git branch and
// worktree. POST /projects/{id}/features/
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
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Determine the base branch (default to the currently checked-out branch).
	if baseBranch == "" {
		current, err := git.CurrentBranch(project.Path)
		if err != nil || current == "" {
			http.Error(w, "could not determine current branch", http.StatusInternalServerError)
			return
		}
		baseBranch = current
	}

	branchName := git.SanitizeBranchName(name)
	worktreePath := git.WorktreeDir(project.Path, branchName)

	// Create the git branch from base.
	if err := git.CreateBranch(project.Path, branchName, baseBranch); err != nil {
		log.Printf("Error creating branch %q: %v", branchName, err)
		http.Error(w, "failed to create branch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Switch back to the base branch so the main worktree isn't on the
	// feature branch, then create a worktree for the feature branch.
	if err := git.CheckoutBranch(project.Path, baseBranch); err != nil {
		log.Printf("Error switching back to base branch %q: %v", baseBranch, err)
	}

	if err := git.CreateWorktree(project.Path, branchName, worktreePath); err != nil {
		log.Printf("Error creating worktree: %v", err)
		http.Error(w, "failed to create worktree: "+err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	featureID := uuid.New().String()

	feature := model.Feature{
		ID:           featureID,
		ProjectID:    project.ID,
		Name:         name,
		BranchName:   branchName,
		BaseBranch:   baseBranch,
		WorktreePath: worktreePath,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.store.AddFeature(feature); err != nil {
		log.Printf("Error storing feature: %v", err)
		http.Error(w, "failed to store feature", http.StatusInternalServerError)
		return
	}

	// Create an initial session in the feature workspace.
	paneID := uuid.New().String()
	sess := model.Session{
		ID:        uuid.New().String(),
		ProjectID: project.ID,
		FeatureID: featureID,
		Name:      "Session " + time.Now().Format("15:04"),
		WorkDir:   worktreePath,
		Layout:    model.NewLeafPane(paneID),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.store.AddSession(sess); err != nil {
		log.Printf("Error creating initial feature session: %v", err)
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/features/"+featureID+"/", http.StatusSeeOther)
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

	// Collect starred projects for quick-switch panel
	var starredProjects []model.Project
	for _, p := range h.store.GetProjects() {
		if p.Starred {
			starredProjects = append(starredProjects, p)
		}
	}

	features := h.store.GetFeatures(project.ID)

	data := map[string]any{
		"Title":           feature.Name + " - " + project.Name + " - ClawIDE",
		"Project":         project,
		"Feature":         feature,
		"Features":        features,
		"Sessions":        sessions,
		"ActiveTab":       "terminal",
		"StarredProjects": starredProjects,
		"ActiveFeatureID": featureID,
		"IsGitRepo":       true,
		"SidebarPosition": h.cfg.SidebarPosition,
		"SidebarWidth":    h.cfg.SidebarWidth,
	}

	if err := h.renderer.RenderHTMX(w, r, "feature-workspace", "feature-workspace", data); err != nil {
		log.Printf("Error rendering feature workspace: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DeleteFeature removes a feature, its worktree, and all associated sessions.
// DELETE /projects/{id}/features/{fid}/
func (h *Handlers) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	featureID := chi.URLParam(r, "fid")

	feature, ok := h.store.GetFeature(featureID)
	if !ok {
		http.Error(w, "feature not found", http.StatusNotFound)
		return
	}

	// Destroy all PTY sessions belonging to this feature.
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

	// Remove the git worktree.
	if err := git.RemoveWorktree(project.Path, feature.WorktreePath); err != nil {
		log.Printf("Error removing worktree %s: %v", feature.WorktreePath, err)
	}

	// Delete the feature (cascades to sessions via store).
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
		Layout:    model.NewLeafPane(paneID),
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
