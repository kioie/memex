// Package memex provides a local-first memory store and MCP server for AI agents.
package memex

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const defaultBusyTimeout = 5 * time.Second

const schema = `
CREATE TABLE IF NOT EXISTS memories (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	tags TEXT NOT NULL DEFAULT '[]',
	memory_type TEXT NOT NULL DEFAULT 'note',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
	content,
	tags,
	content='memories',
	content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
	INSERT INTO memories_fts(rowid, content, tags) VALUES (new.rowid, new.content, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
	INSERT INTO memories_fts(memories_fts, rowid, content, tags) VALUES('delete', old.rowid, old.content, old.tags);
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
	INSERT INTO memories_fts(memories_fts, rowid, content, tags) VALUES('delete', old.rowid, old.content, old.tags);
	INSERT INTO memories_fts(rowid, content, tags) VALUES (new.rowid, new.content, new.tags);
END;
`

// Memory is a single stored fact, preference, decision, or note.
type Memory struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	Tags       []string  `json:"tags,omitempty"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Score      float64   `json:"score,omitempty"`
	Highlights string    `json:"highlights,omitempty"`
}

// Store persists agent memories in a local SQLite database with FTS5 search.
type Store struct {
	db      *sql.DB
	path    string
	writeMu sync.Mutex
}

// DefaultDir returns the default memex data directory (~/.memex).
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".memex"), nil
}

// ResolveDir picks MEMEX_DIR, or falls back to DefaultDir.
func ResolveDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("MEMEX_DIR")); dir != "" {
		return dir, nil
	}
	return DefaultDir()
}

// Open creates or opens a store at dir/memex.db.
func Open(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("memex dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create memex dir: %w", err)
	}
	path := filepath.Join(dir, "memex.db")
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(%d)",
		path, defaultBusyTimeout.Milliseconds())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	return &Store{db: db, path: path}, nil
}

// Path returns the on-disk database file path.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Close closes the underlying database.
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// Remember stores a new memory and returns it with an assigned ID.
func (s *Store) Remember(_ context.Context, content string, tags []string, memoryType string) (*Memory, error) {
	if s == nil {
		return nil, errors.New("store is closed")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("content is required")
	}
	if memoryType == "" {
		memoryType = "note"
	}
	now := time.Now().UTC()
	mem := &Memory{
		ID:        uuid.NewString(),
		Content:   content,
		Tags:      normalizeTags(tags),
		Type:      memoryType,
		CreatedAt: now,
		UpdatedAt: now,
	}
	tagsJSON, err := json.Marshal(mem.Tags)
	if err != nil {
		return nil, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return nil, errors.New("store is closed")
	}
	_, err = s.db.Exec(
		`INSERT INTO memories (id, content, tags, memory_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		mem.ID, mem.Content, string(tagsJSON), mem.Type, mem.CreatedAt.Format(time.RFC3339Nano), mem.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("insert memory: %w", err)
	}
	return mem, nil
}

// Recall searches memories using FTS5. Empty query lists recent memories.
func (s *Store) Recall(_ context.Context, query string, limit int) ([]Memory, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("store is closed")
	}
	if limit <= 0 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return s.listRecent(limit)
	}
	ftsQuery := buildFTSQuery(query)
	rows, err := s.db.Query(`
		SELECT m.id, m.content, m.tags, m.memory_type, m.created_at, m.updated_at, bm25(memories_fts) AS score,
		       snippet(memories_fts, 0, '[', ']', '…', 12) AS highlights
		FROM memories_fts
		JOIN memories m ON m.rowid = memories_fts.rowid
		WHERE memories_fts MATCH ?
		ORDER BY score
		LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()
	return scanMemories(rows)
}

func (s *Store) listRecent(limit int) ([]Memory, error) {
	rows, err := s.db.Query(`
		SELECT id, content, tags, memory_type, created_at, updated_at, 0 AS score, '' AS highlights
		FROM memories
		ORDER BY updated_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	return scanMemories(rows)
}

// Get returns one memory by ID.
func (s *Store) Get(_ context.Context, id string) (*Memory, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("store is closed")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("id is required")
	}
	row := s.db.QueryRow(`
		SELECT id, content, tags, memory_type, created_at, updated_at
		FROM memories WHERE id = ?`, id)
	mem, err := scanMemory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	return mem, err
}

// Forget deletes a memory by ID.
func (s *Store) Forget(_ context.Context, id string) error {
	if s == nil {
		return errors.New("store is closed")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("id is required")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return errors.New("store is closed")
	}
	res, err := s.db.Exec(`DELETE FROM memories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}
	return nil
}

// Stats returns basic store statistics.
func (s *Store) Stats(_ context.Context) (count int, err error) {
	if s == nil || s.db == nil {
		return 0, errors.New("store is closed")
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&count)
	return count, err
}

func scanMemories(rows *sql.Rows) ([]Memory, error) {
	var out []Memory
	for rows.Next() {
		mem, err := scanMemoryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *mem)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMemory(row rowScanner) (*Memory, error) {
	var (
		id, content, tagsJSON, memoryType, createdAt, updatedAt string
	)
	if err := row.Scan(&id, &content, &tagsJSON, &memoryType, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return decodeMemory(id, content, tagsJSON, memoryType, createdAt, updatedAt, 0, "")
}

func scanMemoryRow(rows *sql.Rows) (*Memory, error) {
	var (
		id, content, tagsJSON, memoryType, createdAt, updatedAt, highlights string
		score                                                               float64
	)
	if err := rows.Scan(&id, &content, &tagsJSON, &memoryType, &createdAt, &updatedAt, &score, &highlights); err != nil {
		return nil, err
	}
	return decodeMemory(id, content, tagsJSON, memoryType, createdAt, updatedAt, score, highlights)
}

func decodeMemory(id, content, tagsJSON, memoryType, createdAt, updatedAt string, score float64, highlights string) (*Memory, error) {
	var tags []string
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			return nil, fmt.Errorf("decode tags: %w", err)
		}
	}
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("decode created_at: %w", err)
	}
	updated, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("decode updated_at: %w", err)
	}
	return &Memory{
		ID:         id,
		Content:    content,
		Tags:       tags,
		Type:       memoryType,
		CreatedAt:  created,
		UpdatedAt:  updated,
		Score:      score,
		Highlights: highlights,
	}, nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func buildFTSQuery(query string) string {
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return query
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ReplaceAll(part, `"`, "")
		if part == "" {
			continue
		}
		quoted = append(quoted, `"`+part+`"`)
	}
	return strings.Join(quoted, " OR ")
}
