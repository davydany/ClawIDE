package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/davydany/ccmux/internal/model"
)

type State struct {
	Projects []model.Project `json:"projects"`
	Sessions []model.Session `json:"sessions"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	state    State
}

func New(filePath string) (*Store, error) {
	s := &Store{filePath: filePath}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading state: %w", err)
		}
		s.state = State{}
	}
	return s, nil
}

func (s *Store) GetProjects() []model.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Project, len(s.state.Projects))
	copy(out, s.state.Projects)
	return out
}

func (s *Store) GetProject(id string) (model.Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.state.Projects {
		if p.ID == id {
			return p, true
		}
	}
	return model.Project{}, false
}

func (s *Store) AddProject(p model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Projects = append(s.state.Projects, p)
	return s.save()
}

func (s *Store) UpdateProject(p model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.state.Projects {
		if existing.ID == p.ID {
			s.state.Projects[i] = p
			return s.save()
		}
	}
	return fmt.Errorf("project %s not found", p.ID)
}

func (s *Store) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.state.Projects {
		if p.ID == id {
			s.state.Projects = append(s.state.Projects[:i], s.state.Projects[i+1:]...)
			// Also remove associated sessions
			filtered := s.state.Sessions[:0]
			for _, sess := range s.state.Sessions {
				if sess.ProjectID != id {
					filtered = append(filtered, sess)
				}
			}
			s.state.Sessions = filtered
			return s.save()
		}
	}
	return fmt.Errorf("project %s not found", id)
}

func (s *Store) GetSessions(projectID string) []model.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Session
	for _, sess := range s.state.Sessions {
		if sess.ProjectID == projectID {
			out = append(out, sess)
		}
	}
	return out
}

func (s *Store) GetSession(id string) (model.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.state.Sessions {
		if sess.ID == id {
			return sess, true
		}
	}
	return model.Session{}, false
}

func (s *Store) AddSession(sess model.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Sessions = append(s.state.Sessions, sess)
	return s.save()
}

func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sess := range s.state.Sessions {
		if sess.ID == id {
			s.state.Sessions = append(s.state.Sessions[:i], s.state.Sessions[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("session %s not found", id)
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.state)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}
