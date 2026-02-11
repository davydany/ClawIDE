package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

type SnippetStore struct {
	mu       sync.RWMutex
	filePath string
	snippets []model.Snippet
}

func NewSnippetStore(filePath string) (*SnippetStore, error) {
	s := &SnippetStore{filePath: filePath}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading snippets: %w", err)
		}
		s.snippets = []model.Snippet{}
	}
	return s, nil
}

func (s *SnippetStore) GetAll() []model.Snippet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Snippet, len(s.snippets))
	copy(out, s.snippets)
	return out
}

func (s *SnippetStore) Search(query string) []model.Snippet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var out []model.Snippet
	for _, sn := range s.snippets {
		if strings.Contains(strings.ToLower(sn.Name), q) || strings.Contains(strings.ToLower(sn.Content), q) {
			out = append(out, sn)
		}
	}
	return out
}

func (s *SnippetStore) Get(id string) (model.Snippet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sn := range s.snippets {
		if sn.ID == id {
			return sn, true
		}
	}
	return model.Snippet{}, false
}

func (s *SnippetStore) Add(sn model.Snippet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snippets = append(s.snippets, sn)
	return s.save()
}

func (s *SnippetStore) Update(sn model.Snippet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.snippets {
		if existing.ID == sn.ID {
			s.snippets[i] = sn
			return s.save()
		}
	}
	return fmt.Errorf("snippet %s not found", sn.ID)
}

func (s *SnippetStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sn := range s.snippets {
		if sn.ID == id {
			s.snippets = append(s.snippets[:i], s.snippets[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("snippet %s not found", id)
}

func (s *SnippetStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.snippets)
}

func (s *SnippetStore) save() error {
	data, err := json.MarshalIndent(s.snippets, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling snippets: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}
