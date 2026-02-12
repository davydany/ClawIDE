package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

type BookmarkStore struct {
	mu        sync.RWMutex
	filePath  string
	bookmarks []model.Bookmark
}

func NewBookmarkStore(filePath string) (*BookmarkStore, error) {
	s := &BookmarkStore{filePath: filePath}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading bookmarks: %w", err)
		}
		s.bookmarks = []model.Bookmark{}
	}
	return s, nil
}

func (s *BookmarkStore) GetAll() []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Bookmark, len(s.bookmarks))
	copy(out, s.bookmarks)
	return out
}

// GetByProject returns bookmarks for a project, starred first then alphabetical by name.
func (s *BookmarkStore) GetByProject(projectID string) []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Bookmark
	for _, b := range s.bookmarks {
		if b.ProjectID == projectID {
			out = append(out, b)
		}
	}
	sortBookmarks(out)
	return out
}

// GetStarredByProject returns only starred bookmarks for a project, alphabetical by name.
func (s *BookmarkStore) GetStarredByProject(projectID string) []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Bookmark
	for _, b := range s.bookmarks {
		if b.ProjectID == projectID && b.Starred {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// CountStarredByProject returns the number of starred bookmarks for a project.
func (s *BookmarkStore) CountStarredByProject(projectID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, b := range s.bookmarks {
		if b.ProjectID == projectID && b.Starred {
			count++
		}
	}
	return count
}

// Search performs case-insensitive search on name and URL within a project.
func (s *BookmarkStore) Search(projectID, query string) []model.Bookmark {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var out []model.Bookmark
	for _, b := range s.bookmarks {
		if b.ProjectID != projectID {
			continue
		}
		if strings.Contains(strings.ToLower(b.Name), q) || strings.Contains(strings.ToLower(b.URL), q) {
			out = append(out, b)
		}
	}
	sortBookmarks(out)
	return out
}

func (s *BookmarkStore) Get(id string) (model.Bookmark, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.bookmarks {
		if b.ID == id {
			return b, true
		}
	}
	return model.Bookmark{}, false
}

func (s *BookmarkStore) Add(b model.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bookmarks = append(s.bookmarks, b)
	return s.save()
}

func (s *BookmarkStore) Update(b model.Bookmark) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.bookmarks {
		if existing.ID == b.ID {
			s.bookmarks[i] = b
			return s.save()
		}
	}
	return fmt.Errorf("bookmark %s not found", b.ID)
}

func (s *BookmarkStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, b := range s.bookmarks {
		if b.ID == id {
			s.bookmarks = append(s.bookmarks[:i], s.bookmarks[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("bookmark %s not found", id)
}

func (s *BookmarkStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.bookmarks)
}

func (s *BookmarkStore) save() error {
	data, err := json.MarshalIndent(s.bookmarks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling bookmarks: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}

// sortBookmarks sorts starred first, then alphabetical by name.
func sortBookmarks(bookmarks []model.Bookmark) {
	sort.Slice(bookmarks, func(i, j int) bool {
		if bookmarks[i].Starred != bookmarks[j].Starred {
			return bookmarks[i].Starred
		}
		return strings.ToLower(bookmarks[i].Name) < strings.ToLower(bookmarks[j].Name)
	})
}
