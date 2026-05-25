package memex

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Store) getByHashLocked(userID, hash string) (*Memory, error) {
	row := s.db.QueryRow(`
		SELECT `+sqlMemoryColumns+`
		FROM memories WHERE user_id = ? AND content_hash = ? AND valid_to = ? LIMIT 1`, userID, hash, "")
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return mem, err
}

// UpdateInput carries fields for Update (supersede).
type UpdateInput struct {
	Content  string
	Tags     []string
	Type     string
	Metadata map[string]any
	UserID   string
	AgentID  string
}

// Update supersedes an active memory: marks the old row inactive and appends a new row.
func (s *Store) Update(_ context.Context, id string, in UpdateInput) (*Memory, error) {
	if s == nil {
		return nil, errStoreClosed
	}
	id, err := trimRequired(id, "id")
	if err != nil {
		return nil, err
	}
	content := strings.TrimSpace(in.Content)
	if err := validateContent(content); err != nil {
		return nil, err
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return nil, errStoreClosed
	}

	old, err := s.getActiveLockedForUser(id, ResolveUserIDArg(in.UserID), in.AgentID)
	if err != nil {
		return nil, err
	}
	scopeUserID := old.UserID
	if scopeUserID == "" {
		scopeUserID = defaultUserID
	}
	memoryType := in.Type
	if memoryType == "" {
		memoryType = old.Type
	}
	tags := in.Tags
	if tags == nil {
		tags = old.Tags
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = old.Metadata
	}
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)
	hash := contentHash(scopeUserID, content)
	tagsJSON, err := json.Marshal(normalizeTags(tags))
	if err != nil {
		return nil, err
	}
	metaJSON, err := encodeMetadata(metadata)
	if err != nil {
		return nil, err
	}

	closeQuery := `UPDATE memories SET valid_to = ?, updated_at = ? WHERE id = ? AND user_id = ? AND valid_to = ?`
	closeArgs := []any{nowStr, nowStr, id, scopeUserID, ""}
	if a := ResolveAgentIDArg(in.AgentID); a != "" {
		closeQuery += clauseFilterAgentID
		closeArgs = append(closeArgs, a)
	}
	res, err := s.db.Exec(closeQuery, closeArgs...)
	if err != nil {
		return nil, fmt.Errorf("supersede old memory: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	oldTagsJSON, _ := json.Marshal(old.Tags)
	rowID, err := s.memoryRowIDLocked(id)
	if err != nil {
		return nil, fmt.Errorf("lookup rowid: %w", err)
	}
	if err := s.purgeFTSLocked(rowID, old.Content, string(oldTagsJSON)); err != nil {
		return nil, fmt.Errorf("purge fts: %w", err)
	}

	newID := uuid.NewString()
	mem := &Memory{
		ID:           newID,
		Content:      content,
		Tags:         normalizeTags(tags),
		Type:         memoryType,
		UserID:       scopeUserID,
		AgentID:      old.AgentID,
		RunID:        old.RunID,
		SupersedesID: id,
		Metadata:     metadata,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err = s.db.Exec(
		`INSERT INTO memories (id, content, tags, memory_type, created_at, updated_at, content_hash, metadata, user_id, agent_id, run_id, supersedes_id, valid_to) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mem.ID, mem.Content, string(tagsJSON), mem.Type, nowStr, nowStr, hash, metaJSON, mem.UserID, mem.AgentID, mem.RunID, id, "",
	)
	if err != nil {
		return nil, fmt.Errorf("insert superseding memory: %w", err)
	}
	if err := s.recordHistory(id, scopeUserID, historyUpdate, old.Content, content); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}
	if err := s.recordEvent(id, scopeUserID, eventSupersede, newID, content); err != nil {
		return nil, fmt.Errorf("record event: %w", err)
	}
	if err := s.recordEvent(newID, scopeUserID, eventAdd, id, content); err != nil {
		return nil, fmt.Errorf("record event: %w", err)
	}
	return mem, nil
}

func (s *Store) getLocked(id string) (*Memory, error) {
	return s.getLockedForUser(id, "", "")
}

func (s *Store) getActiveLockedForUser(id, userID, agentID string) (*Memory, error) {
	mem, err := s.getLockedForUser(id, userID, agentID)
	if err != nil {
		return nil, err
	}
	if mem.ValidTo != nil {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	return mem, nil
}

func (s *Store) getLockedForUser(id, userID, agentID string) (*Memory, error) {
	if userID != "" {
		query := `
			SELECT ` + sqlMemoryColumns + `
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
		SELECT `+sqlMemoryColumns+`
		FROM memories WHERE id = ?`, id)
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("memory not found: %s", id)
	}
	return mem, err
}

// List returns memories matching filters without full-text search.
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

func (s *Store) softDeleteLocked(id, userID string, old *Memory) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(
		`UPDATE memories SET valid_to = ?, updated_at = ? WHERE id = ? AND user_id = ? AND valid_to = ?`,
		now, now, id, userID, "",
	)
	if err != nil {
		return fmt.Errorf("soft delete memory: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}
	rowID, err := s.memoryRowIDLocked(id)
	if err != nil {
		return fmt.Errorf("lookup rowid: %w", err)
	}
	tagsJSON, _ := json.Marshal(old.Tags)
	if err := s.purgeFTSLocked(rowID, old.Content, string(tagsJSON)); err != nil {
		return fmt.Errorf("purge fts: %w", err)
	}
	if err := s.recordHistory(id, userID, historyDelete, old.Content, ""); err != nil {
		return err
	}
	return s.recordEvent(id, userID, eventDelete, "", old.Content)
}

// ForgetBatch soft-deletes multiple memories by ID for the scoped user.
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
		if old.UserID != userID || old.ValidTo != nil {
			continue
		}
		if err := s.softDeleteLocked(id, userID, old); err != nil {
			if strings.Contains(err.Error(), "memory not found") {
				continue
			}
			return deleted, err
		}
		deleted++
	}
	if deleted == 0 {
		return 0, errors.New("no memories deleted")
	}
	return deleted, nil
}

// ForgetAll soft-deletes all active memories for a user_id scope.
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

	rows, err := s.db.Query(`SELECT `+sqlMemoryColumns+` FROM memories WHERE user_id = ? AND valid_to = ?`, userID, "")
	if err != nil {
		return 0, err
	}
	var olds []*Memory
	for rows.Next() {
		mem, err := scanMemoryFullRow(rows)
		if err != nil {
			rows.Close()
			return 0, err
		}
		olds = append(olds, mem)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	deleted := 0
	for _, old := range olds {
		if err := s.softDeleteLocked(old.ID, userID, old); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
