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
	ID           string         `json:"id"`
	Content      string         `json:"content"`
	Tags         []string       `json:"tags,omitempty"`
	Type         string         `json:"type"`
	UserID       string         `json:"user_id,omitempty"`
	AgentID      string         `json:"agent_id,omitempty"`
	RunID        string         `json:"run_id,omitempty"`
	SupersedesID string         `json:"supersedes_id,omitempty"`
	ValidTo      *time.Time     `json:"valid_to,omitempty"`
	Source       string         `json:"source,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Score        float64        `json:"score,omitempty"`
	Highlights   string         `json:"highlights,omitempty"`
}

// MemoryFilter scopes list/search operations (local SQLite).
type MemoryFilter struct {
	UserID          string
	AgentID         string
	RunID           string
	Tags            []string
	Type            string
	Source          string
	Metadata        map[string]string
	IncludeInactive bool
	Limit           int
	Offset          int
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
// Duplicate content for the same user_id returns the existing memory (content-hash dedup).
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
	agentID := ResolveAgentIDArg(cfg.AgentID)
	runID := ResolveRunIDArg(cfg.RunID)
	if memoryType == "" {
		memoryType = "note"
	}
	if err := validateMemoryType(memoryType); err != nil {
		return nil, err
	}
	source, err := resolveSource(cfg.Source, memoryType)
	if err != nil {
		return nil, err
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
		AgentID:   agentID,
		RunID:     runID,
		Source:    source,
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
		sqlInsertMemory,
		mem.ID, mem.Content, string(tagsJSON), mem.Type, mem.CreatedAt.Format(time.RFC3339Nano), mem.UpdatedAt.Format(time.RFC3339Nano), hash, metaJSON, mem.UserID, mem.AgentID, mem.RunID, "", "", mem.Source,
	)
	if err != nil {
		return nil, fmt.Errorf("insert memory: %w", err)
	}
	if err := s.recordHistory(mem.ID, userID, historyAdd, "", mem.Content); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}
	if err := s.recordEvent(mem.ID, userID, eventAdd, "", mem.Content); err != nil {
		return nil, fmt.Errorf("record event: %w", err)
	}
	if err := s.indexMemoryRetrievalLocked(mem, tags); err != nil {
		return nil, fmt.Errorf("index retrieval signals: %w", err)
	}
	return mem, nil
}

// Recall searches memories using FTS5. Empty query lists recent memories for the scoped user.
func (s *Store) Recall(ctx context.Context, query string, limit int) ([]Memory, error) {
	return s.Search(ctx, query, MemoryFilter{Limit: limit})
}

// Search runs FTS5 recall with filters (user_id, tags, type, agent_id, run_id, metadata).
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
	return s.hybridSearch(query, filter, limit, max(0, filter.Offset))
}

func (s *Store) listRecentFiltered(limit int, filter MemoryFilter) ([]Memory, error) {
	filter.Limit = limit
	sqlText, args, err := listMemoriesSQL(filter, limit, max(0, filter.Offset))
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	return scanMemoriesFull(rows)
}

// Get returns one memory by ID scoped to userID (defaults to MEMEX_USER_ID).
// When agentID is set (arg or MEMEX_AGENT_ID), the memory must match that agent_id.
func (s *Store) Get(_ context.Context, id, userID, agentID string) (*Memory, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	id, err := trimRequired(id, "id")
	if err != nil {
		return nil, err
	}
	userID = ResolveUserIDArg(userID)
	query := sqlSelectMemoryByIDUser
	args := []any{id, userID}
	if a := ResolveAgentIDArg(agentID); a != "" {
		query += clauseFilterAgentID
		args = append(args, a)
	}
	row := s.db.QueryRow(query, args...)
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errMemoryNotFound(id)
	}
	return mem, err
}

// Forget soft-deletes a memory by ID scoped to userID (defaults to MEMEX_USER_ID).
func (s *Store) Forget(_ context.Context, id, userID, agentID string) error {
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
	old, err := s.getActiveLockedForUser(id, userID, agentID)
	if err != nil {
		return err
	}
	return s.softDeleteLocked(id, userID, agentID, old)
}

// Stats returns basic store statistics.
func (s *Store) Stats(_ context.Context) (count int, err error) {
	if s == nil || s.db == nil {
		return 0, errors.New("store is closed")
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE valid_to = ?`, "").Scan(&count)
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

type memoryScanRow struct {
	id, content, tagsJSON, memoryType, createdAt, updatedAt, metaJSON string
	userID, agentID, runID, supersedesID, validTo, source                      string
	score                                                             float64
	highlights                                                        string
}

func scanMemoryFull(row rowScanner) (*Memory, error) {
	var r memoryScanRow
	if err := row.Scan(&r.id, &r.content, &r.tagsJSON, &r.memoryType, &r.createdAt, &r.updatedAt, &r.metaJSON, &r.userID, &r.agentID, &r.runID, &r.supersedesID, &r.validTo, &r.source); err != nil {
		return nil, err
	}
	return decodeMemoryFull(r)
}

func scanMemoryFullRow(rows *sql.Rows) (*Memory, error) {
	var r memoryScanRow
	if err := rows.Scan(&r.id, &r.content, &r.tagsJSON, &r.memoryType, &r.createdAt, &r.updatedAt, &r.metaJSON, &r.userID, &r.agentID, &r.runID, &r.supersedesID, &r.validTo, &r.source); err != nil {
		return nil, err
	}
	return decodeMemoryFull(r)
}

func scanMemorySearchRow(rows *sql.Rows) (*Memory, error) {
	var r memoryScanRow
	if err := rows.Scan(&r.id, &r.content, &r.tagsJSON, &r.memoryType, &r.createdAt, &r.updatedAt, &r.score, &r.highlights, &r.metaJSON, &r.userID, &r.agentID, &r.runID, &r.supersedesID, &r.validTo, &r.source); err != nil {
		return nil, err
	}
	return decodeMemoryFull(r)
}

func decodeMemoryFull(r memoryScanRow) (*Memory, error) {
	mem, err := decodeMemory(r.id, r.content, r.tagsJSON, r.memoryType, r.createdAt, r.updatedAt, r.score, r.highlights)
	if err != nil {
		return nil, err
	}
	meta, err := decodeMetadata(r.metaJSON)
	if err != nil {
		return nil, err
	}
	mem.Metadata = meta
	mem.UserID = r.userID
	mem.AgentID = r.agentID
	mem.RunID = r.runID
	mem.SupersedesID = r.supersedesID
	mem.Source = normalizeStoredSource(r.source)
	if r.validTo != "" {
		t, err := time.Parse(time.RFC3339Nano, r.validTo)
		if err != nil {
			return nil, fmt.Errorf("decode valid_to: %w", err)
		}
		mem.ValidTo = &t
	}
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
