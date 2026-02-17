package store

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
	"gopkg.in/yaml.v3"
)

// noteIndex is persisted in .clawide/notes/index.yaml and holds folder metadata.
type noteIndex struct {
	Folders []model.Folder `yaml:"folders"`
}

// ProjectNoteStore stores notes as individual .md files with YAML frontmatter
// under <projectDir>/.clawide/notes/.
type ProjectNoteStore struct {
	mu      sync.RWMutex
	baseDir string // .clawide/notes/
	notes   []model.Note
	index   noteIndex
}

// NewProjectNoteStore initialises a project-scoped note store.
// baseDir should point to <project>/.clawide/notes/.
func NewProjectNoteStore(baseDir string) (*ProjectNoteStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating notes dir: %w", err)
	}

	s := &ProjectNoteStore{baseDir: baseDir}
	if err := s.loadIndex(); err != nil {
		// index.yaml is optional; start empty on first use
		s.index = noteIndex{}
	}
	if err := s.loadNotes(); err != nil {
		return nil, fmt.Errorf("loading notes: %w", err)
	}
	// Migrate UUID-named files to title-based filenames
	s.migrateUUIDFilenames()
	return s, nil
}

// --------------- Note CRUD ---------------

func (s *ProjectNoteStore) GetAll() []model.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Note, len(s.notes))
	copy(out, s.notes)
	sortNotesByOrder(out)
	return out
}

func (s *ProjectNoteStore) GetByFolder(folderID string) []model.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Note
	for _, n := range s.notes {
		if n.FolderID == folderID {
			out = append(out, n)
		}
	}
	sortNotesByOrder(out)
	return out
}

func (s *ProjectNoteStore) Search(query string) []model.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var out []model.Note
	for _, n := range s.notes {
		if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
			out = append(out, n)
		}
	}
	sortNotesByOrder(out)
	return out
}

func (s *ProjectNoteStore) Get(id string) (model.Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.notes {
		if n.ID == id {
			return n, true
		}
	}
	return model.Note{}, false
}

func (s *ProjectNoteStore) Add(n model.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasNoteWithTitle(n.Title, n.FolderID, "") {
		return fmt.Errorf("a note with title %q already exists in this folder", n.Title)
	}
	s.notes = append(s.notes, n)
	return s.saveNote(n)
}

func (s *ProjectNoteStore) Update(n model.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.notes {
		if existing.ID == n.ID {
			// Check for duplicate title if title or folder changed
			if n.Title != existing.Title || n.FolderID != existing.FolderID {
				if s.hasNoteWithTitle(n.Title, n.FolderID, n.ID) {
					return fmt.Errorf("a note with title %q already exists in this folder", n.Title)
				}
			}
			oldPath := s.noteFilePath(existing)
			s.notes[i] = n
			newPath := s.noteFilePath(n)
			if oldPath != newPath {
				os.Remove(oldPath)
				// Clean up empty parent directories
				s.cleanEmptyDirs(filepath.Dir(oldPath))
			}
			return s.saveNote(n)
		}
	}
	return fmt.Errorf("note %s not found", n.ID)
}

func (s *ProjectNoteStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.notes {
		if n.ID == id {
			path := s.noteFilePath(n)
			s.notes = append(s.notes[:i], s.notes[i+1:]...)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			s.cleanEmptyDirs(filepath.Dir(path))
			return nil
		}
	}
	return fmt.Errorf("note %s not found", id)
}

func (s *ProjectNoteStore) Reorder(noteIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idxMap := make(map[string]int, len(noteIDs))
	for i, id := range noteIDs {
		idxMap[id] = i
	}
	for i, n := range s.notes {
		if order, ok := idxMap[n.ID]; ok {
			s.notes[i].Order = order
		}
	}
	// persist all reordered notes
	for _, n := range s.notes {
		if _, ok := idxMap[n.ID]; ok {
			if err := s.saveNote(n); err != nil {
				return err
			}
		}
	}
	return nil
}

// HasNoteWithTitle checks if a note with the given title exists in the specified folder.
func (s *ProjectNoteStore) HasNoteWithTitle(title, folderID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hasNoteWithTitle(title, folderID, "")
}

// hasNoteWithTitle is the internal version (caller must hold lock).
// excludeID allows excluding a specific note (for updates).
func (s *ProjectNoteStore) hasNoteWithTitle(title, folderID, excludeID string) bool {
	for _, n := range s.notes {
		if n.Title == title && n.FolderID == folderID && n.ID != excludeID {
			return true
		}
	}
	return false
}

// --------------- Folder CRUD ---------------

func (s *ProjectNoteStore) GetFolders() []model.Folder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Folder, len(s.index.Folders))
	copy(out, s.index.Folders)
	return out
}

func (s *ProjectNoteStore) GetFolder(id string) (model.Folder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.index.Folders {
		if f.ID == id {
			return f, true
		}
	}
	return model.Folder{}, false
}

func (s *ProjectNoteStore) CreateFolder(f model.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := model.ValidateFolderDepth(f.ParentID, s.index.Folders); err != nil {
		return err
	}
	s.index.Folders = append(s.index.Folders, f)
	if err := s.saveIndex(); err != nil {
		return err
	}
	// Create directory on disk
	dirPath := filepath.Join(s.baseDir, s.folderDirPath(f.ID))
	return os.MkdirAll(dirPath, 0755)
}

func (s *ProjectNoteStore) UpdateFolder(f model.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.index.Folders {
		if existing.ID == f.ID {
			if f.ParentID != existing.ParentID {
				if err := model.ValidateFolderDepth(f.ParentID, s.index.Folders); err != nil {
					return err
				}
			}
			// Compute old directory path before updating
			oldDirPath := filepath.Join(s.baseDir, s.folderDirPath(f.ID))

			// Update in memory
			s.index.Folders[i] = f

			// Compute new directory path after updating
			newDirPath := filepath.Join(s.baseDir, s.folderDirPath(f.ID))

			if err := s.saveIndex(); err != nil {
				return err
			}

			// Move directory if path changed
			if oldDirPath != newDirPath {
				// Ensure parent of new path exists
				os.MkdirAll(filepath.Dir(newDirPath), 0755)
				if _, err := os.Stat(oldDirPath); err == nil {
					if err := os.Rename(oldDirPath, newDirPath); err != nil {
						return fmt.Errorf("moving folder directory: %w", err)
					}
				}
				// Clean up empty old parent
				s.cleanEmptyDirs(filepath.Dir(oldDirPath))
			}
			return nil
		}
	}
	return fmt.Errorf("folder %s not found", f.ID)
}

func (s *ProjectNoteStore) DeleteFolder(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the folder and its parent
	var parentID string
	var folderDirRel string
	for _, f := range s.index.Folders {
		if f.ID == id {
			parentID = f.ParentID
			folderDirRel = s.folderDirPath(id)
			break
		}
	}

	folderDir := filepath.Join(s.baseDir, folderDirRel)
	parentDir := filepath.Join(s.baseDir, s.folderDirPath(parentID))

	// Move notes in this folder to parent
	for i, n := range s.notes {
		if n.FolderID == id {
			oldPath := s.noteFilePath(n)
			s.notes[i].FolderID = parentID
			newPath := s.noteFilePath(s.notes[i])
			os.MkdirAll(filepath.Dir(newPath), 0755)
			os.Rename(oldPath, newPath)
		}
	}

	// Re-parent child folders
	for i, f := range s.index.Folders {
		if f.ParentID == id {
			s.index.Folders[i].ParentID = parentID
			// Move child folder directories to parent
			childDirOld := filepath.Join(folderDir, f.Name)
			childDirNew := filepath.Join(parentDir, f.Name)
			if _, err := os.Stat(childDirOld); err == nil {
				os.Rename(childDirOld, childDirNew)
			}
		}
	}

	// Remove the folder from index
	for i, f := range s.index.Folders {
		if f.ID == id {
			s.index.Folders = append(s.index.Folders[:i], s.index.Folders[i+1:]...)
			break
		}
	}

	// Remove the now-empty folder directory
	os.Remove(folderDir)
	s.cleanEmptyDirs(filepath.Dir(folderDir))

	return s.saveIndex()
}

// --------------- Path helpers ---------------

// folderDirPath resolves a folder UUID to its directory path relative to baseDir
// by walking the parent chain.
func (s *ProjectNoteStore) folderDirPath(folderID string) string {
	if folderID == "" {
		return ""
	}
	var parts []string
	current := folderID
	seen := map[string]bool{}
	for current != "" {
		if seen[current] {
			break
		}
		seen[current] = true
		var found bool
		for _, f := range s.index.Folders {
			if f.ID == current {
				parts = append([]string{f.Name}, parts...)
				current = f.ParentID
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return filepath.Join(parts...)
}

// noteFilePath returns the full filesystem path for a note based on its title and folder.
func (s *ProjectNoteStore) noteFilePath(n model.Note) string {
	return filepath.Join(s.baseDir, s.folderDirPath(n.FolderID), n.Title+".md")
}

// cleanEmptyDirs walks up from dir to baseDir, removing empty directories.
func (s *ProjectNoteStore) cleanEmptyDirs(dir string) {
	for dir != s.baseDir && strings.HasPrefix(dir, s.baseDir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// dirPathToFolderID resolves a relative directory path (from baseDir) back to a folder ID.
func (s *ProjectNoteStore) dirPathToFolderID(relDir string) string {
	if relDir == "" || relDir == "." {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(relDir), "/")
	parentID := ""
	for _, part := range parts {
		found := false
		for _, f := range s.index.Folders {
			if f.Name == part && f.ParentID == parentID {
				parentID = f.ID
				found = true
				break
			}
		}
		if !found {
			return "" // directory doesn't match known folder
		}
	}
	return parentID
}

// --------------- Persistence (YAML frontmatter + markdown) ---------------

const frontmatterSep = "---\n"

func (s *ProjectNoteStore) indexPath() string {
	return filepath.Join(s.baseDir, "index.yaml")
}

func (s *ProjectNoteStore) saveNote(n model.Note) error {
	path := s.noteFilePath(n)
	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating note directory: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterSep)
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		return fmt.Errorf("encoding note frontmatter: %w", err)
	}
	enc.Close()
	buf.WriteString(frontmatterSep)
	buf.WriteString(n.Content)
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func (s *ProjectNoteStore) loadNotes() error {
	s.notes = nil
	return filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		n, err := parseNoteFile(data)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		// Derive folder_id from directory structure
		relPath, _ := filepath.Rel(s.baseDir, filepath.Dir(path))
		if relPath != "." && relPath != "" {
			n.FolderID = s.dirPathToFolderID(relPath)
		}
		s.notes = append(s.notes, n)
		return nil
	})
}

func parseNoteFile(data []byte) (model.Note, error) {
	content := string(data)

	// Strip leading "---\n"
	if !strings.HasPrefix(content, frontmatterSep) {
		return model.Note{}, fmt.Errorf("missing opening frontmatter separator")
	}
	content = content[len(frontmatterSep):]

	// Find closing "---\n"
	idx := strings.Index(content, frontmatterSep)
	if idx < 0 {
		return model.Note{}, fmt.Errorf("missing closing frontmatter separator")
	}

	frontmatter := content[:idx]
	body := content[idx+len(frontmatterSep):]

	var n model.Note
	if err := yaml.Unmarshal([]byte(frontmatter), &n); err != nil {
		return model.Note{}, fmt.Errorf("decoding frontmatter: %w", err)
	}
	n.Content = body

	return n, nil
}

func (s *ProjectNoteStore) loadIndex() error {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &s.index)
}

func (s *ProjectNoteStore) saveIndex() error {
	data, err := yaml.Marshal(&s.index)
	if err != nil {
		return fmt.Errorf("encoding note index: %w", err)
	}
	return os.WriteFile(s.indexPath(), data, 0644)
}

func sortNotesByOrder(notes []model.Note) {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Order < notes[j].Order
	})
}

// --------------- Migration ---------------

var uuidFilenameRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.md$`)

func (s *ProjectNoteStore) migrateUUIDFilenames() {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !uuidFilenameRegex.MatchString(entry.Name()) {
			continue
		}
		// Find the note in memory by ID (filename without .md)
		noteID := strings.TrimSuffix(entry.Name(), ".md")
		for i, n := range s.notes {
			if n.ID == noteID {
				oldPath := filepath.Join(s.baseDir, entry.Name())
				// Sanitize title if needed
				title := n.Title
				if model.ValidateNoteTitle(title) != nil {
					title = sanitizeTitle(title)
					s.notes[i].Title = title
				}
				newPath := s.noteFilePath(s.notes[i])
				if oldPath != newPath {
					os.MkdirAll(filepath.Dir(newPath), 0755)
					if err := os.Rename(oldPath, newPath); err != nil {
						log.Printf("notes migration: failed to rename %s -> %s: %v", oldPath, newPath, err)
						continue
					}
					// Re-save to update frontmatter
					s.saveNote(s.notes[i])
					log.Printf("notes migration: renamed %s -> %s", entry.Name(), filepath.Base(newPath))
				}
				break
			}
		}
	}
}

func sanitizeTitle(title string) string {
	var result []byte
	for i := 0; i < len(title); i++ {
		c := title[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-' {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	s := string(result)
	if s == "" || s == "." || s == ".." {
		s = "untitled"
	}
	return s
}
