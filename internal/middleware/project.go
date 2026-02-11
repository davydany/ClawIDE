package middleware

import (
	"context"
	"net/http"

	"github.com/davydany/ccmux/internal/model"
	"github.com/davydany/ccmux/internal/store"
	"github.com/go-chi/chi/v5"
)

const projectKey contextKey = "project"

func ProjectLoader(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			projectID := chi.URLParam(r, "id")
			if projectID == "" {
				http.Error(w, "project ID required", http.StatusBadRequest)
				return
			}

			project, ok := st.GetProject(projectID)
			if !ok {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), projectKey, project)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetProject(r *http.Request) model.Project {
	p, _ := r.Context().Value(projectKey).(model.Project)
	return p
}
