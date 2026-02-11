package handler

import (
	"log"
	"net/http"
)

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	projects := h.store.GetProjects()

	data := map[string]any{
		"Title":    "CCMux - Dashboard",
		"Projects": projects,
	}

	if err := h.renderer.RenderHTMX(w, r, "project-list", "project-list", data); err != nil {
		log.Printf("Error rendering dashboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
