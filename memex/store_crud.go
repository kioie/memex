package memex

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Store) getByHashLocked(userID, hash string) (*Memory, error) {
	row := s.db.QueryRow(`
		SELECT id, content, tags, memory_type, created_at, updated_at, metadata, user_id, agent_id, run_id
		FROM memories WHERE user_id = ? AND content_hash = ? LIMIT 1`, userID, hash)
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return mem, err
}

// Update overwrites memory content scoped to userID (defaults to MEMEX_USER_ID).
func (s *Store) Update(_ context.Context, id string, content string, tags []string, memoryType string, metadata map[string]any, userID, agentID string) (*Memory, error) {
	if s == nil {
		return nil, errStoreClosed
	}
	id, err := trimRequired(id, "id")
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if err := validateContent(content); err != nil {
		return nil, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return nil, errStoreClosed
	}

	old, err := s.getLockedForUser(id, ResolveUserIDArg(userID), agentID)
	if err != nil {
		return nil, err
	}
	scopeUserID := old.UserID
	if scopeUserID == "" {
		scopeUserID = defaultUserID
	}
	if memoryType == "" {
		memoryType = old.Type
	}
	if tags == nil {
		tags = old.Tags
	}
	if metadata == nil {
		metadata = old.Metadata
	}
	now := time.Now().UTC()
	hash := contentHash(scopeUserID, content)
	tagsJSON, err := json.Marshal(normalizeTags(tags))
	if err != nil {
		return nil, err
	}
	metaJSON, err := encodeMetadata(metadata)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE memories SET content = ?, tags = ?, memory_type = ?, updated_at = ?, content_hash = ?, metadata = ?
		WHERE id = ? AND user_id = ?`
	args := []any{content, string(tagsJSON), memoryType, now.Format(time.RFC3339Nano), hash, metaJSON, id, scopeUserID}
	if a := ResolveAgentIDArg(agentID); a != "" {
		query += clauseFilterAgentID
		args = append(args, a)
	}
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("update memory: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	if err := s.recordHistory(id, scopeUserID, historyUpdate, old.Content, content); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}
	return &Memory{
		ID:        id,
		Content:   content,
		Tags:      normalizeTags(tags),
		Type:      memoryType,
		UserID:    scopeUserID,
		AgentID:   old.AgentID,
		RunID:     old.RunID,
		Metadata:  metadata,
		CreatedAt: old.CreatedAt,
		UpdatedAt: now,
	}, nil
}

func (s *Store) getLocked(id string) (*Memory, error) {
	return s.getLockedForUser(id, "", "")
}

func (s *Store) getLockedForUser(id, userID, agentID string) (*Memory, error) {
	if userID != "" {
		query := `
			SELECT id, content, tags, memory_type, created_at, updated_at, metadata, user_id, agent_id, run_id
			FROM memories WHERE id = ? AND user_id = ?`
		args := []any{id, userID}
		if a := ResolveAgentIDArg(agentID); a != "" {
			query += clauseFilterAgentID
			args = append(args, a)
		}
		row := s.db.QueryRow(query, args...)
		mem, err := scanMemoryFull(row)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("memory not found: %s", id)
		}
		return mem, err
	}
	row := s.db.QueryRow(`
		SELECT id, content, tags, memory_type, created_at, updated_at, metadata, user_id, agent_id, run_id
		FROM memories WHERE id = ?`, id)
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	return mem, err
}

// List returns memories matching filters without full-text search (mem0 get_all).
func (s *Store) List(_ context.Context, filter MemoryFilter) ([]Memory, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := max(0, filter.Offset)

	sqlText, args, err := listMemoriesSQL(filter, limit, offset)
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

// ForgetBatch deletes multiple memories by ID for the scoped user.
func (s *Store) ForgetBatch(_ context.Context, ids []string, userID string) (int, error) {
	if s == nil {
		return 0, errStoreClosed
	}
	userID = ResolveUserIDArg(userID)
	if len(ids) == 0 {
		return 0, errors.New("ids is required")
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return 0, errStoreClosed
	}

	deleted := 0
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		old, err := s.getLocked(id)
		if err != nil {
			if strings.Contains(err.Error(), "memory not found") {
				continue
			}
			return deleted, err
		}
		if old.UserID != userID {
			continue
		}
		res, err := s.db.Exec(`DELETE FROM memories WHERE id = ? AND user_id = ?`, id, userID)
		if err != nil {
			return deleted, fmt.Errorf("delete memory: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			continue
		}
		if err := s.recordHistory(id, userID, historyDelete, old.Content, ""); err != nil {
			return deleted, fmt.Errorf("record history: %w", err)
		}
		deleted++
	}
	if deleted == 0 {
		return 0, errors.New("no memories deleted")
	}
	return deleted, nil
}

// ForgetAll deletes all memories for a user_id scope (mem0 delete_all).
func (s *Store) ForgetAll(_ context.Context, userID string) (int, error) {
	if s == nil {
		return 0, errStoreClosed
	}
	userID = ResolveUserIDArg(userID)

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return 0, errStoreClosed
	}

	rows, err := s.db.Query(`SELECT id, content FROM memories WHERE user_id = ?`, userID)
	if err != nil {
		return 0, err
	}
	type pair struct{ id, content string }
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.id, &p.content); err != nil {
			rows.Close()
			return 0, err
		}
		pairs = append(pairs, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	res, err := s.db.Exec(`DELETE FROM memories WHERE user_id = ?`, userID)
	if err != nil {
		return 0, fmt.Errorf("delete all: %w", err)
	}
	n, _ := res.RowsAffected()
	for _, p := range pairs {
		_ = s.recordHistory(p.id, userID, historyDelete, p.content, "")
	}
	return int(n), nil
}
