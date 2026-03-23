package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/skills"
	"github.com/go-chi/chi/v5"
)

func (h *Handlers) resolveSkillsDir(r *http.Request, scope string) string {
	switch scope {
	case "global":
		return skills.GlobalSkillsDir()
	case "project":
		project := middleware.GetProject(r)
		return skills.ProjectSkillsDir(project.Path)
	default:
		return ""
	}
}

// ListSkills returns all skills (global + project), optionally filtered by scope query param.
func (h *Handlers) ListSkills(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	scopeFilter := r.URL.Query().Get("scope")

	globalDir := skills.GlobalSkillsDir()
	projectDir := skills.ProjectSkillsDir(project.Path)

	if scopeFilter == "global" {
		projectDir = ""
	} else if scopeFilter == "project" {
		globalDir = ""
	}

	all, err := skills.ListSkills(globalDir, projectDir)
	if err != nil {
		log.Printf("Error listing skills: %v", err)
		http.Error(w, "Failed to list skills", http.StatusInternalServerError)
		return
	}

	if all == nil {
		all = []skills.Skill{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all)
}

// GetSkill returns a single skill by scope and directory name.
func (h *Handlers) GetSkill(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	skillName := chi.URLParam(r, "skillName")

	baseDir := h.resolveSkillsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	skill, err := skills.GetSkill(baseDir, skillName)
	if err != nil {
		log.Printf("Error getting skill %s/%s: %v", scope, skillName, err)
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	skill.Scope = scope
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(skill)
}

// CreateSkill creates a new skill in the specified scope.
func (h *Handlers) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var skill skills.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if skill.Name == "" {
		http.Error(w, "Skill name is required", http.StatusBadRequest)
		return
	}

	scope := skill.Scope
	if scope == "" {
		scope = "project"
	}

	baseDir := h.resolveSkillsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := skills.CreateSkill(baseDir, skill); err != nil {
		log.Printf("Error creating skill: %v", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// UpdateSkill updates an existing skill.
func (h *Handlers) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	skillName := chi.URLParam(r, "skillName")

	baseDir := h.resolveSkillsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	var skill skills.Skill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := skills.UpdateSkill(baseDir, skillName, skill); err != nil {
		log.Printf("Error updating skill %s/%s: %v", scope, skillName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// DeleteSkill removes a skill directory.
func (h *Handlers) DeleteSkill(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	skillName := chi.URLParam(r, "skillName")

	baseDir := h.resolveSkillsDir(r, scope)
	if baseDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := skills.DeleteSkill(baseDir, skillName); err != nil {
		log.Printf("Error deleting skill %s/%s: %v", scope, skillName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// MoveSkill moves a skill between global and project scope.
func (h *Handlers) MoveSkill(w http.ResponseWriter, r *http.Request) {
	scope := chi.URLParam(r, "scope")
	skillName := chi.URLParam(r, "skillName")

	var body struct {
		TargetScope string `json:"target_scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if body.TargetScope != "global" && body.TargetScope != "project" {
		http.Error(w, "target_scope must be 'global' or 'project'", http.StatusBadRequest)
		return
	}
	if body.TargetScope == scope {
		http.Error(w, "Skill is already in that scope", http.StatusBadRequest)
		return
	}

	srcDir := h.resolveSkillsDir(r, scope)
	dstDir := h.resolveSkillsDir(r, body.TargetScope)
	if srcDir == "" || dstDir == "" {
		http.Error(w, "Invalid scope", http.StatusBadRequest)
		return
	}

	if err := skills.MoveSkill(srcDir, dstDir, skillName); err != nil {
		log.Printf("Error moving skill %s/%s to %s: %v", scope, skillName, body.TargetScope, err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "moved", "new_scope": body.TargetScope})
}
