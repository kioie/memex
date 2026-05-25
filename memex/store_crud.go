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
	row := s.db.QueryRow(sqlSelectMemoryByHash, userID, hash, "")
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
	Source   string
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
	in.Content = content
	if in.Type != "" {
		if err := validateMemoryType(in.Type); err != nil {
			return nil, err
		}
	}
	if in.Source != "" {
		if err := validateSource(in.Source); err != nil {
			return nil, err
		}
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.db == nil {
		return nil, errStoreClosed
	}
	return s.supersedeLocked(id, in)
}

type supersedePayload struct {
	scopeUserID string
	content     string
	memoryType  string
	source      string
	tags        []string
	metadata    map[string]any
	tagsJSON    string
	metaJSON    string
	hash        string
	now         time.Time
	nowStr      string
}

func prepareSupersedePayload(old *Memory, in UpdateInput) (*supersedePayload, error) {
	scopeUserID := old.UserID
	if scopeUserID == "" {
		scopeUserID = defaultUserID
	}
	memoryType := in.Type
	if memoryType == "" {
		memoryType = old.Type
	}
	if err := validateMemoryType(memoryType); err != nil {
		return nil, err
	}
	source := in.Source
	if source == "" {
		source = old.Source
	}
	if err := validateSource(source); err != nil {
		return nil, err
	}
	tags := in.Tags
	if tags == nil {
		tags = old.Tags
	}
	metadata := in.Metadata
	if metadata == nil {
		metadata = old.Metadata
	}
	tagsJSON, err := json.Marshal(normalizeTags(tags))
	if err != nil {
		return nil, err
	}
	metaJSON, err := encodeMetadata(metadata)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &supersedePayload{
		scopeUserID: scopeUserID,
		content:     in.Content,
		memoryType:  memoryType,
		source:      source,
		tags:        tags,
		metadata:    metadata,
		tagsJSON:    string(tagsJSON),
		metaJSON:    metaJSON,
		hash:        contentHash(scopeUserID, in.Content),
		now:         now,
		nowStr:      now.Format(time.RFC3339Nano),
	}, nil
}

func (s *Store) closeSupersededRowLocked(id, scopeUserID, agentID, nowStr string, old *Memory) error {
	if err := s.setInactiveLocked(id, scopeUserID, agentID, nowStr); err != nil {
		return fmt.Errorf("supersede old memory: %w", err)
	}
	if err := s.purgeMemoryFromFTSLocked(id, old); err != nil {
		return fmt.Errorf("purge fts: %w", err)
	}
	return nil
}

func (s *Store) supersedeLocked(id string, in UpdateInput) (*Memory, error) {
	old, err := s.getActiveLockedForUser(id, ResolveUserIDArg(in.UserID), in.AgentID)
	if err != nil {
		return nil, err
	}
	payload, err := prepareSupersedePayload(old, in)
	if err != nil {
		return nil, err
	}
	if err := s.closeSupersededRowLocked(id, payload.scopeUserID, in.AgentID, payload.nowStr, old); err != nil {
		return nil, err
	}

	newID := uuid.NewString()
	mem := &Memory{
		ID:           newID,
		Content:      payload.content,
		Tags:         normalizeTags(payload.tags),
		Type:         payload.memoryType,
		UserID:       payload.scopeUserID,
		AgentID:      old.AgentID,
		RunID:        old.RunID,
		SupersedesID: id,
		Source:       payload.source,
		Metadata:     payload.metadata,
		CreatedAt:    payload.now,
		UpdatedAt:    payload.now,
	}
	_, err = s.db.Exec(
		sqlInsertMemory,
		mem.ID, mem.Content, payload.tagsJSON, mem.Type, payload.nowStr, payload.nowStr, payload.hash, payload.metaJSON, mem.UserID, mem.AgentID, mem.RunID, id, "", mem.Source,
	)
	if err != nil {
		return nil, fmt.Errorf("insert superseding memory: %w", err)
	}
	if err := s.recordHistory(id, payload.scopeUserID, historyUpdate, old.Content, payload.content); err != nil {
		return nil, fmt.Errorf("record history: %w", err)
	}
	if err := s.recordEvent(id, payload.scopeUserID, eventSupersede, newID, payload.content); err != nil {
		return nil, fmt.Errorf("record event: %w", err)
	}
	if err := s.recordEvent(newID, payload.scopeUserID, eventAdd, id, payload.content); err != nil {
		return nil, fmt.Errorf("record event: %w", err)
	}
	if err := s.indexMemoryRetrievalLocked(mem, payload.tags); err != nil {
		return nil, fmt.Errorf("index retrieval signals: %w", err)
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
		return nil, errMemoryNotFound(id)
	}
	return mem, nil
}

func (s *Store) getLockedForUser(id, userID, agentID string) (*Memory, error) {
	if userID != "" {
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
	row := s.db.QueryRow(sqlSelectMemoryByID, id)
	mem, err := scanMemoryFull(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errMemoryNotFound(id)
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

func (s *Store) softDeleteLocked(id, userID, agentID string, old *Memory) error {
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.setInactiveLocked(id, userID, agentID, nowStr); err != nil {
		return fmt.Errorf("soft delete memory: %w", err)
	}
	if err := s.purgeMemoryFromFTSLocked(id, old); err != nil {
		return err
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
		if err := s.softDeleteLocked(id, userID, "", old); err != nil {
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

	rows, err := s.db.Query(sqlSelectActiveByUser, userID, "")
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
		if err := s.softDeleteLocked(old.ID, userID, "", old); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}
