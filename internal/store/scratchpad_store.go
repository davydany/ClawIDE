package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
)

type ScratchpadStore struct {
	mu       sync.RWMutex
	filePath string
	pad      model.Scratchpad
}

func NewScratchpadStore(filePath string) (*ScratchpadStore, error) {
	s := &ScratchpadStore{filePath: filePath}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading scratchpad: %w", err)
		}
		s.pad = model.Scratchpad{}
	}
	return s, nil
}

func (s *ScratchpadStore) Get() model.Scratchpad {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pad
}

func (s *ScratchpadStore) Update(content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pad.Content = content
	s.pad.UpdatedAt = time.Now()
	return s.save()
}

func (s *ScratchpadStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.pad)
}

func (s *ScratchpadStore) save() error {
	data, err := json.MarshalIndent(s.pad, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling scratchpad: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}
