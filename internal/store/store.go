package store

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/lsy88/uptime-chopper/internal/model"
)

type State struct {
	Monitors      []model.Monitor      `json:"monitors"`
	Notifications []model.Notification `json:"notifications"`
}

type Store interface {
	GetState() State
	UpsertMonitor(m model.Monitor) (model.Monitor, error)
	DeleteMonitor(id string) error

	GetNotifications() []model.Notification
	UpsertNotification(n model.Notification) (model.Notification, error)
	DeleteNotification(id string) error

	AddMonitorHistory(id string, entry model.MonitorHistoryEntry) error
	GetMonitorHistory(id string) ([]model.MonitorHistoryEntry, error)
	PruneMonitorHistory(id string, days int) error
}

type JSONStore struct {
	filePath string
	mu       sync.RWMutex
	state    State
	history  map[string][]model.MonitorHistoryEntry
}

func NewJSONStore(filePath string) (*JSONStore, error) {
	s := &JSONStore{
		filePath: filePath,
		history:  make(map[string][]model.MonitorHistoryEntry),
	}
	if err := s.load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.state = State{
				Monitors:      []model.Monitor{},
				Notifications: []model.Notification{},
			}
			return s, s.persist()
		}
		return nil, err
	}
	// Ensure non-nil slices
	if s.state.Monitors == nil {
		s.state.Monitors = []model.Monitor{}
	}
	if s.state.Notifications == nil {
		s.state.Notifications = []model.Notification{}
	}
	return s, nil
}

func (s *JSONStore) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *JSONStore) UpsertMonitor(m model.Monitor) (model.Monitor, error) {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i := range s.state.Monitors {
		if s.state.Monitors[i].ID == m.ID {
			m.CreatedAt = s.state.Monitors[i].CreatedAt
			m.UpdatedAt = now
			s.state.Monitors[i] = m
			found = true
			break
		}
	}

	if !found {
		m.CreatedAt = now
		m.UpdatedAt = now
		s.state.Monitors = append(s.state.Monitors, m)
	}

	if err := s.persistLocked(); err != nil {
		return model.Monitor{}, err
	}

	return m, nil
}

func (s *JSONStore) DeleteMonitor(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dst := s.state.Monitors[:0]
	for _, m := range s.state.Monitors {
		if m.ID == id {
			continue
		}
		dst = append(dst, m)
	}
	s.state.Monitors = dst

	return s.persistLocked()
}

func (s *JSONStore) GetNotifications() []model.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy
	dst := make([]model.Notification, len(s.state.Notifications))
	copy(dst, s.state.Notifications)
	return dst
}

func (s *JSONStore) UpsertNotification(n model.Notification) (model.Notification, error) {
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i := range s.state.Notifications {
		if s.state.Notifications[i].ID == n.ID {
			n.CreatedAt = s.state.Notifications[i].CreatedAt
			n.UpdatedAt = now
			s.state.Notifications[i] = n
			found = true
			break
		}
	}

	if !found {
		n.CreatedAt = now
		n.UpdatedAt = now
		s.state.Notifications = append(s.state.Notifications, n)
	}

	if err := s.persistLocked(); err != nil {
		return model.Notification{}, err
	}

	return n, nil
}

func (s *JSONStore) DeleteNotification(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dst := s.state.Notifications[:0]
	for _, n := range s.state.Notifications {
		if n.ID == id {
			continue
		}
		dst = append(dst, n)
	}
	s.state.Notifications = dst

	return s.persistLocked()
}

func (s *JSONStore) AddMonitorHistory(id string, entry model.MonitorHistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hist := s.history[id]
	// Prepend
	hist = append([]model.MonitorHistoryEntry{entry}, hist...)
	// Keep last 50
	if len(hist) > 50 {
		hist = hist[:50]
	}
	s.history[id] = hist
	return nil
}

func (s *JSONStore) GetMonitorHistory(id string) ([]model.MonitorHistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hist := s.history[id]
	if hist == nil {
		return []model.MonitorHistoryEntry{}, nil
	}
	// Return copy
	out := make([]model.MonitorHistoryEntry, len(hist))
	copy(out, hist)
	return out, nil
}

func (s *JSONStore) load() error {
	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}

	s.state = st
	return nil
}

func (s *JSONStore) persist() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistLocked()
}

func (s *JSONStore) persistLocked() error {
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}
