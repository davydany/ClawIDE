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

	"github.com/davydany/ccmux/internal/middleware"
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

// ListFiles handles GET /projects/{id}/api/files?path=...
// It returns a JSON array of FileEntry objects for the requested directory.
// Query parameters:
//   - path: relative path within the project (default ".")
//   - hidden: if "true", include hidden files (dotfiles)
func (h *Handlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}

	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		requestedPath = "."
	}
	showHidden := r.URL.Query().Get("hidden") == "true"

	absPath, ok := resolveAndValidatePath(project.Path, requestedPath)
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

	cleanRoot := filepath.Clean(project.Path)
	var files []FileEntry
	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files unless explicitly requested.
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		entryInfo, err := entry.Info()
		if err != nil {
			continue // skip entries we cannot stat
		}

		// Build the relative path from the project root for the response.
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

	// Sort: directories first, then alphabetical by name (case-insensitive).
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Return empty array instead of null when there are no entries.
	if files == nil {
		files = []FileEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// ReadFile handles GET /projects/{id}/api/file?path=...
// It reads a file from the project directory and returns its contents.
// Text files are served as text/plain; binary files as application/octet-stream.
// Files larger than 1MB are rejected.
func (h *Handlers) ReadFile(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}

	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	absPath, ok := resolveAndValidatePath(project.Path, requestedPath)
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

	// Read the file content (capped at maxFileReadSize).
	data, err := io.ReadAll(io.LimitReader(f, maxFileReadSize+1))
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	contentType := detectContentType(data)
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

// WriteFile handles PUT /projects/{id}/api/file?path=...
// It writes the request body to the specified file within the project directory.
func (h *Handlers) WriteFile(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}

	requestedPath := r.URL.Query().Get("path")
	if requestedPath == "" {
		http.Error(w, "path parameter is required", http.StatusBadRequest)
		return
	}

	absPath, ok := resolveAndValidatePath(project.Path, requestedPath)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Ensure the parent directory exists.
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
