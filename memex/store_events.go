package memex

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	eventAdd       = "ADD"
	eventSupersede = "SUPERSEDE"
	eventDelete    = "DELETE"
)

const eventsSchema = `
CREATE TABLE IF NOT EXISTS memory_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	memory_id TEXT NOT NULL,
	user_id TEXT NOT NULL DEFAULT 'default',
	event_type TEXT NOT NULL,
	related_id TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL DEFAULT '',
	occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memory_events_memory_id ON memory_events(memory_id);
`

func (s *Store) recordEvent(memoryID, userID, eventType, relatedID, content string) error {
	_, err := s.db.Exec(
		`INSERT INTO memory_events (memory_id, user_id, event_type, related_id, content, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`,
		memoryID, userID, eventType, relatedID, content, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) purgeFTSLocked(rowID int64, content, tagsJSON string) error {
	_, err := s.db.Exec(
		`INSERT INTO memories_fts(memories_fts, rowid, content, tags) VALUES('delete', ?, ?, ?)`,
		rowID, content, tagsJSON,
	)
	return err
}

func (s *Store) setInactiveLocked(id, userID, agentID, nowStr string) error {
	query := `UPDATE memories SET valid_to = ?, updated_at = ? WHERE id = ? AND user_id = ? AND valid_to = ?`
	args := []any{nowStr, nowStr, id, userID, ""}
	if a := ResolveAgentIDArg(agentID); a != "" {
		query += clauseFilterAgentID
		args = append(args, a)
	}
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("deactivate memory: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errMemoryNotFound(id)
	}
	return nil
}

func (s *Store) purgeMemoryFromFTSLocked(id string, old *Memory) error {
	rowID, err := s.memoryRowIDLocked(id)
	if err != nil {
		return fmt.Errorf("lookup rowid: %w", err)
	}
	tagsJSON, _ := json.Marshal(old.Tags)
	if err := s.purgeFTSLocked(rowID, old.Content, string(tagsJSON)); err != nil {
		return fmt.Errorf("purge fts: %w", err)
	}
	return nil
}

func (s *Store) memoryRowIDLocked(id string) (int64, error) {
	var rowID int64
	err := s.db.QueryRow(`SELECT rowid FROM memories WHERE id = ?`, id).Scan(&rowID)
	return rowID, err
}
