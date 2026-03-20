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

// mkdirForRoot is the shared implementation for creating a directory under a
// given root path. It creates all intermediate directories as needed.
func mkdirForRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
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

	if err := os.MkdirAll(absPath, 0755); err != nil {
		http.Error(w, "failed to create directory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// renameForRoot renames a file or directory. Both `path` (old) and `newPath`
// (new) are resolved relative to rootPath. The destination must not exist.
func renameForRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
	oldRel := r.URL.Query().Get("path")
	newRel := r.URL.Query().Get("newPath")
	if oldRel == "" || newRel == "" {
		http.Error(w, "path and newPath parameters are required", http.StatusBadRequest)
		return
	}

	absOld, ok := resolveAndValidatePath(rootPath, oldRel)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	absNew, ok := resolveAndValidatePath(rootPath, newRel)
	if !ok {
		http.Error(w, "invalid newPath", http.StatusBadRequest)
		return
	}

	cleanRoot := filepath.Clean(rootPath)
	if absOld == cleanRoot {
		http.Error(w, "cannot rename root directory", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(absOld); os.IsNotExist(err) {
		http.Error(w, "source path not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(absNew); err == nil {
		http.Error(w, "destination already exists", http.StatusConflict)
		return
	}

	// Ensure the destination parent directory exists.
	if err := os.MkdirAll(filepath.Dir(absNew), 0755); err != nil {
		http.Error(w, "failed to create destination directory", http.StatusInternalServerError)
		return
	}

	if err := os.Rename(absOld, absNew); err != nil {
		http.Error(w, "failed to rename: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"oldPath": oldRel,
		"newPath": newRel,
	})
}

// deleteForRoot deletes a file or directory under rootPath.
func deleteForRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
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

	cleanRoot := filepath.Clean(rootPath)
	if absPath == cleanRoot {
		http.Error(w, "cannot delete root directory", http.StatusBadRequest)
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

	if info.IsDir() {
		err = os.RemoveAll(absPath)
	} else {
		err = os.Remove(absPath)
	}
	if err != nil {
		http.Error(w, "failed to delete: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// skipDirs contains directory names that searchFilesForRoot should not descend into.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
}

// searchFilesForRoot walks the directory tree under rootPath and returns files
// whose names match the query. The query is first tried as a glob pattern via
// filepath.Match; if that fails it falls back to a case-insensitive substring
// match on the full relative path. Results are capped at 50 entries.
func searchFilesForRoot(w http.ResponseWriter, r *http.Request, rootPath string) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "q parameter is required", http.StatusBadRequest)
		return
	}

	showHidden := r.URL.Query().Get("hidden") == "true"
	cleanRoot := filepath.Clean(rootPath)
	qLower := strings.ToLower(q)

	var results []FileEntry
	const maxResults = 50

	filepath.WalkDir(cleanRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		name := d.Name()

		// Skip hidden files/dirs unless requested.
		if !showHidden && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip known heavy directories.
		if d.IsDir() && skipDirs[name] {
			return filepath.SkipDir
		}

		// Don't include the root itself.
		if path == cleanRoot {
			return nil
		}

		// Try glob match on filename first.
		matched, globErr := filepath.Match(q, name)
		if globErr != nil {
			// Invalid glob pattern — fall back to substring only.
			matched = false
		}

		relPath, relErr := filepath.Rel(cleanRoot, path)
		if relErr != nil {
			return nil
		}

		if !matched {
			// Case-insensitive substring on the full relative path.
			if !strings.Contains(strings.ToLower(relPath), qLower) {
				return nil
			}
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}

		results = append(results, FileEntry{
			Name:     name,
			Path:     relPath,
			IsDir:    d.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime().UTC(),
		})
		return nil
	})

	if results == nil {
		results = []FileEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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

// Mkdir handles POST /projects/{id}/api/mkdir?path=...
func (h *Handlers) Mkdir(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	mkdirForRoot(w, r, project.Path)
}

// RenameFile handles POST /projects/{id}/api/rename?path=...&newPath=...
func (h *Handlers) RenameFile(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	renameForRoot(w, r, project.Path)
}

// DeleteFile handles DELETE /projects/{id}/api/file?path=...
func (h *Handlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	deleteForRoot(w, r, project.Path)
}

// SearchFiles handles GET /projects/{id}/api/files/search?q=...
func (h *Handlers) SearchFiles(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r)
	if project.Path == "" {
		http.Error(w, "project path not configured", http.StatusInternalServerError)
		return
	}
	searchFilesForRoot(w, r, project.Path)
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
