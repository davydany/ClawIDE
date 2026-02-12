package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/davydany/ClawIDE/internal/model"
)

type NotificationStore struct {
	mu               sync.RWMutex
	filePath         string
	maxNotifications int
	notifications    []model.Notification
}

func NewNotificationStore(filePath string, maxNotifications int) (*NotificationStore, error) {
	s := &NotificationStore{
		filePath:         filePath,
		maxNotifications: maxNotifications,
	}
	if err := s.load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("loading notifications: %w", err)
		}
		s.notifications = []model.Notification{}
	}
	return s, nil
}

func (s *NotificationStore) GetAll() []model.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Notification, len(s.notifications))
	copy(out, s.notifications)
	return out
}

func (s *NotificationStore) GetUnread() []model.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Notification
	for _, n := range s.notifications {
		if !n.Read {
			out = append(out, n)
		}
	}
	return out
}

func (s *NotificationStore) UnreadCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, n := range s.notifications {
		if !n.Read {
			count++
		}
	}
	return count
}

func (s *NotificationStore) Get(id string) (model.Notification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.notifications {
		if n.ID == id {
			return n, true
		}
	}
	return model.Notification{}, false
}

func (s *NotificationStore) Add(n model.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotency dedup: if IdempotencyKey is set, skip if already exists
	if n.IdempotencyKey != "" {
		for _, existing := range s.notifications {
			if existing.IdempotencyKey == n.IdempotencyKey {
				return nil
			}
		}
	}

	// Prepend (newest first)
	s.notifications = append([]model.Notification{n}, s.notifications...)

	// Auto-prune to max
	if len(s.notifications) > s.maxNotifications {
		s.notifications = s.notifications[:s.maxNotifications]
	}

	return s.save()
}

func (s *NotificationStore) MarkRead(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.notifications {
		if n.ID == id {
			s.notifications[i].Read = true
			return s.save()
		}
	}
	return fmt.Errorf("notification %s not found", id)
}

func (s *NotificationStore) MarkAllRead() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.notifications {
		s.notifications[i].Read = true
	}
	return s.save()
}

func (s *NotificationStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, n := range s.notifications {
		if n.ID == id {
			s.notifications = append(s.notifications[:i], s.notifications[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("notification %s not found", id)
}

func (s *NotificationStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.notifications)
}

func (s *NotificationStore) save() error {
	data, err := json.MarshalIndent(s.notifications, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling notifications: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}
