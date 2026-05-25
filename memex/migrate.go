package memex

import (
	"database/sql"
	"fmt"
)

const historySchema = `
CREATE TABLE IF NOT EXISTS memory_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	memory_id TEXT NOT NULL,
	user_id TEXT NOT NULL DEFAULT 'default',
	action TEXT NOT NULL,
	old_content TEXT NOT NULL DEFAULT '',
	new_content TEXT NOT NULL DEFAULT '',
	changed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memory_history_memory_id ON memory_history(memory_id);
`

const (
	colTextNotNullEmpty       = "TEXT NOT NULL DEFAULT ''"
	colTextNotNullDefaultJSON = "TEXT NOT NULL DEFAULT '{}'"
	colTextNotNullDefaultUser = "TEXT NOT NULL DEFAULT 'default'"
)

func migrateStore(db *sql.DB) error {
	columns := map[string]string{
		"content_hash": colTextNotNullEmpty,
		"metadata":     colTextNotNullDefaultJSON,
		"user_id":      colTextNotNullDefaultUser,
		"agent_id":     colTextNotNullEmpty,
		"run_id":       colTextNotNullEmpty,
	}
	for name, def := range columns {
		exists, err := columnExists(db, "memories", name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE memories ADD COLUMN %s %s", name, def)); err != nil {
			return fmt.Errorf("add column %s: %w", name, err)
		}
	}
	if _, err := db.Exec(historySchema); err != nil {
		return fmt.Errorf("migrate history: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_user_hash ON memories(user_id, content_hash)`); err != nil {
		return fmt.Errorf("create dedup index: %w", err)
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_memories_user_agent ON memories(user_id, agent_id)`); err != nil {
		return fmt.Errorf("create agent scope index: %w", err)
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
