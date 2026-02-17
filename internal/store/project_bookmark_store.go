package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

// bookmarkData is the on-disk structure for .clawide/bookmarks/bookmarks.json.
type bookmarkData struct {
	Folders   []model.Folder   `json:"folders"`
	Bookmarks []model.Bookmark `json:"bookmarks"`
}

// ProjectBookmarkStore stores bookmarks and their folder hierarchy in a single
// JSON file at <project>/.clawide/bookmarks/bookmarks.json.
type ProjectBookmarkStore struct {
	mu      sync.RWMutex
	baseDir string // .clawide/bookmarks/
	data    bookmarkData
}

// NewProjectBookmarkStore initialises a project-scoped bookmark store.
func NewProjectBookmarkStore(baseDir string) (*ProjectBookmarkStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating bookmarks dir: %w", err)
	}

	s := &ProjectBookmarkStore{baseDir: baseDir}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading bookmarks: %w", err)
		}
		s.data = bookmarkData{}
	}
	return s, nil
}

// --------------- Bookmark CRUD ---------------

func (s *ProjectBookmarkStore) GetAll() []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Bookmark, len(s.data.Bookmarks))
	copy(out, s.data.Bookmarks)
	sortProjectBookmarks(out)
	return out
}

func (s *ProjectBookmarkStore) GetByFolder(folderID string) []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Bookmark
	for _, b := range s.data.Bookmarks {
		if b.FolderID == folderID {
			out = append(out, b)
		}
	}
	sortProjectBookmarks(out)
	return out
}

func (s *ProjectBookmarkStore) GetInBar() []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Bookmark
	for _, b := range s.data.Bookmarks {
		if b.InBar {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Order < out[j].Order
	})
	return out
}

func (s *ProjectBookmarkStore) CountInBar() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, b := range s.data.Bookmarks {
		if b.InBar {
			count++
		}
	}
	return count
}

func (s *ProjectBookmarkStore) Search(query string) []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var out []model.Bookmark
	for _, b := range s.data.Bookmarks {
		if strings.Contains(strings.ToLower(b.Name), q) || strings.Contains(strings.ToLower(b.URL), q) {
			out = append(out, b)
		}
	}
	sortProjectBookmarks(out)
	return out
}

func (s *ProjectBookmarkStore) Get(id string) (model.Bookmark, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.data.Bookmarks {
		if b.ID == id {
			return b, true
		}
	}
	return model.Bookmark{}, false
}

func (s *ProjectBookmarkStore) Add(b model.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Bookmarks = append(s.data.Bookmarks, b)
	return s.save()
}

func (s *ProjectBookmarkStore) Update(b model.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Bookmarks {
		if existing.ID == b.ID {
			s.data.Bookmarks[i] = b
			return s.save()
		}
	}
	return fmt.Errorf("bookmark %s not found", b.ID)
}

func (s *ProjectBookmarkStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, b := range s.data.Bookmarks {
		if b.ID == id {
			s.data.Bookmarks = append(s.data.Bookmarks[:i], s.data.Bookmarks[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("bookmark %s not found", id)
}

func (s *ProjectBookmarkStore) Reorder(bookmarkIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idxMap := make(map[string]int, len(bookmarkIDs))
	for i, id := range bookmarkIDs {
		idxMap[id] = i
	}
	for i, b := range s.data.Bookmarks {
		if order, ok := idxMap[b.ID]; ok {
			s.data.Bookmarks[i].Order = order
		}
	}
	return s.save()
}

// --------------- Folder CRUD ---------------

func (s *ProjectBookmarkStore) GetFolders() []model.Folder {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Folder, len(s.data.Folders))
	copy(out, s.data.Folders)
	return out
}

func (s *ProjectBookmarkStore) GetFolder(id string) (model.Folder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.data.Folders {
		if f.ID == id {
			return f, true
		}
	}
	return model.Folder{}, false
}

func (s *ProjectBookmarkStore) CreateFolder(f model.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := model.ValidateFolderDepth(f.ParentID, s.data.Folders); err != nil {
		return err
	}
	s.data.Folders = append(s.data.Folders, f)
	return s.save()
}

func (s *ProjectBookmarkStore) UpdateFolder(f model.Folder) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.data.Folders {
		if existing.ID == f.ID {
			if f.ParentID != existing.ParentID {
				if err := model.ValidateFolderDepth(f.ParentID, s.data.Folders); err != nil {
					return err
				}
			}
			s.data.Folders[i] = f
			return s.save()
		}
	}
	return fmt.Errorf("folder %s not found", f.ID)
}

func (s *ProjectBookmarkStore) DeleteFolder(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Move bookmarks in this folder to root
	for i, b := range s.data.Bookmarks {
		if b.FolderID == id {
			s.data.Bookmarks[i].FolderID = ""
		}
	}

	// Re-parent child folders
	var parentID string
	for _, f := range s.data.Folders {
		if f.ID == id {
			parentID = f.ParentID
			break
		}
	}
	for i, f := range s.data.Folders {
		if f.ParentID == id {
			s.data.Folders[i].ParentID = parentID
		}
	}

	// Remove
	for i, f := range s.data.Folders {
		if f.ID == id {
			s.data.Folders = append(s.data.Folders[:i], s.data.Folders[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("folder %s not found", id)
}

// --------------- Persistence ---------------

func (s *ProjectBookmarkStore) filePath() string {
	return filepath.Join(s.baseDir, "bookmarks.json")
}

func (s *ProjectBookmarkStore) load() error {
	data, err := os.ReadFile(s.filePath())
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

func (s *ProjectBookmarkStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling bookmarks: %w", err)
	}
	return os.WriteFile(s.filePath(), data, 0644)
}

// sortProjectBookmarks sorts by InBar first (bar items on top), then by order, then alphabetical.
func sortProjectBookmarks(bookmarks []model.Bookmark) {
	sort.Slice(bookmarks, func(i, j int) bool {
		if bookmarks[i].InBar != bookmarks[j].InBar {
			return bookmarks[i].InBar
		}
		if bookmarks[i].Order != bookmarks[j].Order {
			return bookmarks[i].Order < bookmarks[j].Order
		}
		return strings.ToLower(bookmarks[i].Name) < strings.ToLower(bookmarks[j].Name)
	})
}
