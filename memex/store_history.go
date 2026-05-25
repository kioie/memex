package memex

import (
	"context"
	"fmt"
	"time"
)

const (
	historyAdd    = "ADD"
	historyUpdate = "UPDATE"
	historyDelete = "DELETE"
)

// HistoryEntry is one audit row for a memory change.
type HistoryEntry struct {
	ID         int64     `json:"id"`
	MemoryID   string    `json:"memory_id"`
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	OldContent string    `json:"old_content,omitempty"`
	NewContent string    `json:"new_content,omitempty"`
	ChangedAt  time.Time `json:"changed_at"`
}

func (s *Store) recordHistory(memoryID, userID, action, oldContent, newContent string) error {
	_, err := s.db.Exec(
		`INSERT INTO memory_history (memory_id, user_id, action, old_content, new_content, changed_at) VALUES (?, ?, ?, ?, ?, ?)`,
		memoryID, userID, action, oldContent, newContent, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

// History returns change records for a memory scoped to userID, newest first.
func (s *Store) History(_ context.Context, id, userID string) ([]HistoryEntry, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	id, err := trimRequired(id, "id")
	if err != nil {
		return nil, err
	}
	userID = ResolveUserIDArg(userID)
	if _, err := s.getLockedForUser(id, userID, ""); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
		SELECT id, memory_id, user_id, action, old_content, new_content, changed_at
		FROM memory_history
		WHERE memory_id = ?
		ORDER BY id DESC`, id)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()
	var out []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var changedAt string
		if err := rows.Scan(&e.ID, &e.MemoryID, &e.UserID, &e.Action, &e.OldContent, &e.NewContent, &changedAt); err != nil {
			return nil, err
		}
		e.ChangedAt, err = time.Parse(time.RFC3339Nano, changedAt)
		if err != nil {
			return nil, fmt.Errorf("decode changed_at: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
