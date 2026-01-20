package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lsy88/uptime-chopper/internal/model"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewSQLiteStore(filePath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite db: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return s, nil
}

func (s *SQLiteStore) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS monitors (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS monitor_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			monitor_id TEXT NOT NULL,
			status TEXT NOT NULL,
			checked_at DATETIME NOT NULL,
			latency_ms INTEGER NOT NULL,
			message TEXT,
			logs TEXT,
			FOREIGN KEY(monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_history_monitor_id_checked_at ON monitor_history(monitor_id, checked_at DESC);`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to exec query %q: %w", query, err)
		}
	}

	// Migration for existing tables
	s.ensureColumns()

	return nil
}

func (s *SQLiteStore) ensureColumns() {
	// Add logs column to monitor_history if it doesn't exist
	// We can try to query it, if fails, add it.
	// Or just try to add it and ignore error "duplicate column name"
	_, err := s.db.Exec("ALTER TABLE monitor_history ADD COLUMN logs TEXT")
	if err != nil {
		// Ignore error, likely column already exists
	}
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := State{
		Monitors:      []model.Monitor{},
		Notifications: []model.Notification{},
	}

	// Load Monitors
	rows, err := s.db.Query("SELECT data FROM monitors")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var data string
			if err := rows.Scan(&data); err == nil {
				var m model.Monitor
				if err := json.Unmarshal([]byte(data), &m); err == nil {
					state.Monitors = append(state.Monitors, m)
				}
			}
		}
	}

	// Load Notifications
	nRows, err := s.db.Query("SELECT data FROM notifications")
	if err == nil {
		defer nRows.Close()
		for nRows.Next() {
			var data string
			if err := nRows.Scan(&data); err == nil {
				var n model.Notification
				if err := json.Unmarshal([]byte(data), &n); err == nil {
					state.Notifications = append(state.Notifications, n)
				}
			}
		}
	}

	return state
}

func (s *SQLiteStore) UpsertMonitor(m model.Monitor) (model.Monitor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	m.UpdatedAt = now
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}

	data, err := json.Marshal(m)
	if err != nil {
		return model.Monitor{}, err
	}

	// Check if exists to preserve CreatedAt if not set (though we handle it above, best to be safe)
	// Actually we should read existing to keep CreatedAt if it's already there and m.CreatedAt is zero?
	// But usually m passed here has CreatedAt if it's an update.
	// Let's just use upsert logic.

	query := `INSERT INTO monitors (id, data, created_at, updated_at) VALUES (?, ?, ?, ?)
			  ON CONFLICT(id) DO UPDATE SET data=excluded.data, updated_at=excluded.updated_at`

	_, err = s.db.Exec(query, m.ID, string(data), m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return model.Monitor{}, err
	}

	return m, nil
}

func (s *SQLiteStore) DeleteMonitor(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM monitors WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) GetNotifications() []model.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notifications := []model.Notification{}
	rows, err := s.db.Query("SELECT data FROM notifications")
	if err != nil {
		return notifications
	}
	defer rows.Close()

	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err == nil {
			var n model.Notification
			if err := json.Unmarshal([]byte(data), &n); err == nil {
				notifications = append(notifications, n)
			}
		}
	}
	return notifications
}

func (s *SQLiteStore) UpsertNotification(n model.Notification) (model.Notification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	n.UpdatedAt = now
	if n.CreatedAt.IsZero() {
		n.CreatedAt = now
	}

	data, err := json.Marshal(n)
	if err != nil {
		return model.Notification{}, err
	}

	query := `INSERT INTO notifications (id, data, created_at, updated_at) VALUES (?, ?, ?, ?)
			  ON CONFLICT(id) DO UPDATE SET data=excluded.data, updated_at=excluded.updated_at`

	_, err = s.db.Exec(query, n.ID, string(data), n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return model.Notification{}, err
	}

	return n, nil
}

func (s *SQLiteStore) DeleteNotification(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM notifications WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) AddMonitorHistory(id string, entry model.MonitorHistoryEntry) error {
	// Not locking strictly needed for INSERT, but let's keep it safe if we add logic later
	// s.mu.Lock()
	// defer s.mu.Unlock()

	query := `INSERT INTO monitor_history (monitor_id, status, checked_at, latency_ms, message, logs) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, id, string(entry.Status), entry.CheckedAt, entry.LatencyMs, entry.Message, entry.Logs)
	return err
}

func (s *SQLiteStore) GetMonitorHistory(id string) ([]model.MonitorHistoryEntry, error) {
	// s.mu.RLock()
	// defer s.mu.RUnlock()

	// Get last 50 entries
	query := `SELECT status, checked_at, latency_ms, message, logs FROM monitor_history WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 50`
	rows, err := s.db.Query(query, id)
	if err != nil {
		return []model.MonitorHistoryEntry{}, err
	}
	defer rows.Close()

	var history []model.MonitorHistoryEntry
	for rows.Next() {
		var entry model.MonitorHistoryEntry
		var status string
		var logs sql.NullString
		if err := rows.Scan(&status, &entry.CheckedAt, &entry.LatencyMs, &entry.Message, &logs); err != nil {
			continue
		}
		entry.Status = model.MonitorStatus(status)
		if logs.Valid {
			entry.Logs = logs.String
		}
		history = append(history, entry)
	}

	// Reverse to match expected order (oldest first? or newest first?)
	// Engine logic was prepending: append([]...{entry}, hist...) -> newest at 0.
	// But typical UI expects time series. MonitorDetail.tsx does `[...history].reverse()`.
	// Engine `GetHistory` returns `hist` which has newest at 0.
	// So `history[0]` is latest.
	// My SQL query returns DESC (newest first), so history[0] is latest.
	// This matches Engine behavior.

	return history, nil
}

func (s *SQLiteStore) PruneMonitorHistory(id string, days int) error {
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	query := `DELETE FROM monitor_history WHERE monitor_id = ? AND checked_at < ?`
	_, err := s.db.Exec(query, id, cutoff)
	return err
}

func (s *SQLiteStore) MigrateFromJSON(jsonPath string) error {
	js, err := NewJSONStore(jsonPath)
	if err != nil {
		// If file doesn't exist, nothing to migrate
		return nil
	}

	state := js.GetState()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we already have data
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM monitors").Scan(&count); err == nil && count > 0 {
		return nil // Already have data, skip migration
	}

	for _, m := range state.Monitors {
		// Upsert logic duplicated here or call UpsertMonitor (but UpsertMonitor locks)
		// Since we hold lock, we should use internal method or just do it here.
		// Calling UpsertMonitor would deadlock.
		// Let's copy logic.
		now := time.Now().UTC()
		if m.CreatedAt.IsZero() {
			m.CreatedAt = now
		}
		if m.UpdatedAt.IsZero() {
			m.UpdatedAt = now
		}
		data, _ := json.Marshal(m)
		query := `INSERT INTO monitors (id, data, created_at, updated_at) VALUES (?, ?, ?, ?)`
		if _, err := s.db.Exec(query, m.ID, string(data), m.CreatedAt, m.UpdatedAt); err != nil {
			return err
		}
	}

	for _, n := range state.Notifications {
		now := time.Now().UTC()
		if n.CreatedAt.IsZero() {
			n.CreatedAt = now
		}
		if n.UpdatedAt.IsZero() {
			n.UpdatedAt = now
		}
		data, _ := json.Marshal(n)
		query := `INSERT INTO notifications (id, data, created_at, updated_at) VALUES (?, ?, ?, ?)`
		if _, err := s.db.Exec(query, n.ID, string(data), n.CreatedAt, n.UpdatedAt); err != nil {
			return err
		}
	}

	return nil
}
