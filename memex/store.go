// Package memex provides a local-first memory store and MCP server for AI agents.
//
// Persistence and search live here; the MCP wire protocol (stdio, tool schemas, handler
// registration) is delegated to github.com/kioie/tiny-go-mcp-server/tinymcp — see
// server.go for tool handlers and tinymcp godoc for RegisterTool / TextResult / Start.
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

// maxMemoryContentLen caps stored fact size to limit context-poisoning and runaway writes.
const maxMemoryContentLen = 256 << 10 // 256 KiB

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
	ID         string         `json:"id"`
	Content    string         `json:"content"`
	Tags       []string       `json:"tags,omitempty"`
	Type       string         `json:"type"`
	UserID     string         `json:"user_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Score      float64        `json:"score,omitempty"`
	Highlights string         `json:"highlights,omitempty"`
}

// MemoryFilter scopes list/search operations (mem0-style filters, local SQLite).
type MemoryFilter struct {
	UserID string
	Tags   []string
	Type   string
	Limit  int
	Offset int
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
	return resolveDataDir(filepath.Join(home, ".memex"))
}

// ResolveDir is implemented in store_path.go.

// Open creates or opens a store at dir/memex.db.
func Open(dir string) (*Store, error) {
	absDir, err := resolveDataDir(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("create memex dir: %w", err)
	}
	path := filepath.Join(absDir, "memex.db")
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
	if err := migrateStore(db); err != nil {
		_ = db.Close()
		return nil, err
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
// Duplicate content for the same user_id returns the existing memory (mem0 hash dedup, storage-only).
func (s *Store) Remember(_ context.Context, content string, tags []string, memoryType string, opts ...RememberOption) (*Memory, error) {
	if s == nil {
		return nil, errStoreClosed
	}
	content = strings.TrimSpace(content)
	if err := validateContent(content); err != nil {
		return nil, err
	}
	cfg := applyRememberOptions(opts)
	userID := ResolveUserIDArg(cfg.UserID)
	if memoryType == "" {
		memoryType = "note"
	}
	hash := contentHash(userID, content)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return nil, errStoreClosed
	}

	if existing, err := s.getByHashLocked(userID, hash); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	now := time.Now().UTC()
	mem := &Memory{
		ID:        uuid.NewString(),
		Content:   content,
		Tags:      normalizeTags(tags),
		Type:      memoryType,
		UserID:    userID,
		Metadata:  cfg.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	tagsJSON, err := json.Marshal(mem.Tags)
	if err != nil {
		return nil, err
	}
	metaJSON, err := encodeMetadata(mem.Metadata)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(
		`INSERT INTO memories (id, content, tags, memory_type, created_at, updated_at, content_hash, metadata, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mem.ID, mem.Content, string(tagsJSON), mem.Type, mem.CreatedAt.Format(time.RFC3339Nano), mem.UpdatedAt.Format(time.RFC3339Nano), hash, metaJSON, mem.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert memory: %w", err)
	}
	if err := s.recordHistory(mem.ID, userID, historyAdd, "", mem.Content); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}
	return mem, nil
}

// Recall searches memories using FTS5. Empty query lists recent memories for the scoped user.
func (s *Store) Recall(ctx context.Context, query string, limit int) ([]Memory, error) {
	return s.Search(ctx, query, MemoryFilter{Limit: limit})
}

// Search runs FTS5 recall with mem0-style filters (user_id, tags, type).
func (s *Store) Search(_ context.Context, query string, filter MemoryFilter) ([]Memory, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return s.listRecentFiltered(limit, filter)
	}
	ftsQuery := buildFTSQuery(query)
	sqlText, args := searchMemoriesSQL(ftsQuery, filter, limit, max(0, filter.Offset))
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()
	return scanMemoriesSearch(rows)
}

func (s *Store) listRecentFiltered(limit int, filter MemoryFilter) ([]Memory, error) {
	filter.Limit = limit
	sqlText, args := listMemoriesSQL(filter, limit, max(0, filter.Offset))
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	return scanMemoriesFull(rows)
}

// Get returns one memory by ID scoped to userID (defaults to MEMEX_USER_ID).
func (s *Store) Get(_ context.Context, id, userID string) (*Memory, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	id, err := trimRequired(id, "id")
	if err != nil {
		return nil, err
	}
	userID = ResolveUserIDArg(userID)
	row := s.db.QueryRow(`
		SELECT id, content, tags, memory_type, created_at, updated_at, metadata, user_id
		FROM memories WHERE id = ? AND user_id = ?`, id, userID)
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	return mem, err
}

// Forget deletes a memory by ID scoped to userID (defaults to MEMEX_USER_ID).
func (s *Store) Forget(_ context.Context, id, userID string) error {
	if s == nil {
		return errStoreClosed
	}
	id, err := trimRequired(id, "id")
	if err != nil {
		return err
	}
	userID = ResolveUserIDArg(userID)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return errStoreClosed
	}
	old, err := s.getLockedForUser(id, userID)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`DELETE FROM memories WHERE id = ? AND user_id = ?`, id, userID)
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
	return s.recordHistory(id, userID, historyDelete, old.Content, "")
}

// Stats returns basic store statistics.
func (s *Store) Stats(_ context.Context) (count int, err error) {
	if s == nil || s.db == nil {
		return 0, errors.New("store is closed")
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&count)
	return count, err
}

func scanMemoriesSearch(rows *sql.Rows) ([]Memory, error) {
	var out []Memory
	for rows.Next() {
		mem, err := scanMemorySearchRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *mem)
	}
	return out, rows.Err()
}

func scanMemoriesFull(rows *sql.Rows) ([]Memory, error) {
	var out []Memory
	for rows.Next() {
		mem, err := scanMemoryFullRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *mem)
	}
	return out, rows.Err()
}

func scanMemoryFull(row rowScanner) (*Memory, error) {
	var (
		id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON, userID string
	)
	if err := row.Scan(&id, &content, &tagsJSON, &memoryType, &createdAt, &updatedAt, &metaJSON, &userID); err != nil {
		return nil, err
	}
	return decodeMemoryFull(id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON, userID, 0, "")
}

func scanMemoryFullRow(rows *sql.Rows) (*Memory, error) {
	var (
		id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON, userID string
	)
	if err := rows.Scan(&id, &content, &tagsJSON, &memoryType, &createdAt, &updatedAt, &metaJSON, &userID); err != nil {
		return nil, err
	}
	return decodeMemoryFull(id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON, userID, 0, "")
}

func scanMemorySearchRow(rows *sql.Rows) (*Memory, error) {
	var (
		id, content, tagsJSON, memoryType, createdAt, updatedAt, highlights, metaJSON, userID string
		score                                                                               float64
	)
	if err := rows.Scan(&id, &content, &tagsJSON, &memoryType, &createdAt, &updatedAt, &score, &highlights, &metaJSON, &userID); err != nil {
		return nil, err
	}
	return decodeMemoryFull(id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON, userID, score, highlights)
}

func decodeMemoryFull(id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON, userID string, score float64, highlights string) (*Memory, error) {
	mem, err := decodeMemory(id, content, tagsJSON, memoryType, createdAt, updatedAt, score, highlights)
	if err != nil {
		return nil, err
	}
	meta, err := decodeMetadata(metaJSON)
	if err != nil {
		return nil, err
	}
	mem.Metadata = meta
	mem.UserID = userID
	if mem.UserID == "" {
		mem.UserID = defaultUserID
	}
	return mem, nil
}

type rowScanner interface {
	Scan(dest ...any) error
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
	// Each token is quoted for FTS5 MATCH; the full string is always passed as a bound
	// query parameter (never concatenated into SQL text).
	parts := strings.Fields(query)
	if len(parts) == 0 {
		return query
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if token := sanitizeFTSToken(part); token != "" {
			quoted = append(quoted, `"`+token+`"`)
		}
	}
	if len(quoted) == 0 {
		return `""`
	}
	return strings.Join(quoted, " OR ")
}

func sanitizeFTSToken(token string) string {
	token = strings.Trim(token, `"`)
	var b strings.Builder
	for _, r := range token {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		if r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
