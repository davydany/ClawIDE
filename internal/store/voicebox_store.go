package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

type VoiceBoxStore struct {
	mu       sync.RWMutex
	filePath string
	maxItems int
	entries  []model.VoiceBoxEntry
}

func NewVoiceBoxStore(filePath string, maxItems int) (*VoiceBoxStore, error) {
	s := &VoiceBoxStore{filePath: filePath, maxItems: maxItems}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading voicebox entries: %w", err)
		}
		s.entries = []model.VoiceBoxEntry{}
	}
	return s, nil
}

func (s *VoiceBoxStore) GetAll() []model.VoiceBoxEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.VoiceBoxEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *VoiceBoxStore) Add(entry model.VoiceBoxEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Prepend newest first
	s.entries = append([]model.VoiceBoxEntry{entry}, s.entries...)
	// Auto-prune to max
	if len(s.entries) > s.maxItems {
		s.entries = s.entries[:s.maxItems]
	}
	return s.save()
}

func (s *VoiceBoxStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if e.ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("voicebox entry %s not found", id)
}

func (s *VoiceBoxStore) DeleteAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = []model.VoiceBoxEntry{}
	return s.save()
}

func (s *VoiceBoxStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.entries)
}

func (s *VoiceBoxStore) save() error {
	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling voicebox entries: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}
