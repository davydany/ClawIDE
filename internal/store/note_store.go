package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

type NoteStore struct {
	mu       sync.RWMutex
	filePath string
	notes    []model.Note
}

func NewNoteStore(filePath string) (*NoteStore, error) {
	s := &NoteStore{filePath: filePath}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading notes: %w", err)
		}
		s.notes = []model.Note{}
	}
	return s, nil
}

func (s *NoteStore) GetAll() []model.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Note, len(s.notes))
	copy(out, s.notes)
	return out
}

func (s *NoteStore) GetByProject(projectID string) []model.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Note
	for _, n := range s.notes {
		if n.ProjectID == projectID {
			out = append(out, n)
		}
	}
	return out
}

func (s *NoteStore) Search(projectID, query string) []model.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	var out []model.Note
	for _, n := range s.notes {
		if n.ProjectID != projectID {
			continue
		}
		if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
			out = append(out, n)
		}
	}
	return out
}

func (s *NoteStore) Get(id string) (model.Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.notes {
		if n.ID == id {
			return n, true
		}
	}
	return model.Note{}, false
}

func (s *NoteStore) Add(n model.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notes = append(s.notes, n)
	return s.save()
}

func (s *NoteStore) Update(n model.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.notes {
		if existing.ID == n.ID {
			s.notes[i] = n
			return s.save()
		}
	}
	return fmt.Errorf("note %s not found", n.ID)
}

func (s *NoteStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.notes {
		if n.ID == id {
			s.notes = append(s.notes[:i], s.notes[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("note %s not found", id)
}

func (s *NoteStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.notes)
}

func (s *NoteStore) save() error {
	data, err := json.MarshalIndent(s.notes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling notes: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}
