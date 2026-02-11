package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/middleware"
)

const maxFileReadSize = 1 << 20 // 1MB

// FileEntry represents a single file or directory in a listing.
type FileEntry struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	IsDir    bool      `json:"is_dir"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// resolveAndValidatePath resolves the requested path relative to the project
// root and ensures it does not escape outside the root via traversal (e.g. "..").
// Returns the cleaned absolute path or an empty string if validation fails.
func resolveAndValidatePath(projectRoot, requestedPath string) (string, bool) {
	// Clean the project root to get a canonical form.
	cleanRoot := filepath.Clean(projectRoot)

	// Resolve the requested path relative to the project root.
	resolved := filepath.Join(cleanRoot, filepath.Clean("/"+requestedPath))
	resolved = filepath.Clean(resolved)

	// The resolved path must be equal to the root or nested within it.
	// We check with a prefix + separator to avoid matching partial directory
	// names (e.g. /project-root-extra should not match /project-root).
	if resolved == cleanRoot {
		return resolved, true
	}
	if strings.HasPrefix(resolved, cleanRoot+string(filepath.Separator)) {
		return resolved, true
	}

	return "", false
}

// listFilesForRoot is the shared implementation for listing directory contents
// under a given root path. Both the project and feature file browsers delegate
// to this function.
func listFilesForRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "."
	}
	showHidden := r.URL.Query().Get("hidden") == "true"

	absPath, ok := resolveAndValidatePath(rootPath, requestedPath)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "path not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to stat path", http.StatusInternalServerError)
		return
	}
	if !info.IsDir() {
		http.Error(w, "path is not a directory", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		http.Error(w, "failed to read directory", http.StatusInternalServerError)
		return
	}

	cleanRoot := filepath.Clean(rootPath)
	var files []FileEntry
	for _, entry := range entries {
		name := entry.Name()

		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}

		entryAbsPath := filepath.Join(absPath, name)
		relPath, err := filepath.Rel(cleanRoot, entryAbsPath)
		if err != nil {
			continue
		}

		files = append(files, FileEntry{
			Name:     name,
			Path:     relPath,
			IsDir:    entry.IsDir(),
			Size:     entryInfo.Size(),
			Modified: entryInfo.ModTime().UTC(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	if files == nil {
		files = []FileEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// readFileFromRoot is the shared implementation for reading a file under a
// given root path.
func readFileFromRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	absPath, ok := resolveAndValidatePath(rootPath, requestedPath)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to stat file", http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.Error(w, "path is a directory, not a file", http.StatusBadRequest)
		return
	}
	if info.Size() > maxFileReadSize {
		http.Error(w, "file too large (max 1MB)", http.StatusRequestEntityTooLarge)
		return
	}

	f, err := os.Open(absPath)
	if err != nil {
		http.Error(w, "failed to open file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxFileReadSize+1))
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	contentType := detectContentType(data)
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// writeFileToRoot is the shared implementation for writing a file under a
// given root path.
func writeFileToRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	absPath, ok := resolveAndValidatePath(rootPath, requestedPath)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, "failed to create directory", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := os.WriteFile(absPath, body, 0644); err != nil {
		http.Error(w, "failed to write file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListFiles handles GET /projects/{id}/api/files?path=...
func (h *Handlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	listFilesForRoot(w, r, project.Path)
}

// ReadFile handles GET /projects/{id}/api/file?path=...
func (h *Handlers) ReadFile(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	readFileFromRoot(w, r, project.Path)
}

// WriteFile handles PUT /projects/{id}/api/file?path=...
func (h *Handlers) WriteFile(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	writeFileToRoot(w, r, project.Path)
}

// detectContentType determines whether content is text or binary.
// It uses a simple heuristic: if the first 512 bytes contain a null byte,
// it is treated as binary.
func detectContentType(data []byte) string {
	// Check up to the first 512 bytes for null bytes.
	checkLen := 512
	if len(data) < checkLen {
		checkLen = len(data)
	}
	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return "application/octet-stream"
		}
	}
	return "text/plain; charset=utf-8"
}
