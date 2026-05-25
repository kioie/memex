package memex

import (
	"context"
	"fmt"
)

// DoctorReport summarizes store health and effective defaults for operators and agents.
type DoctorReport struct {
	Version              string `json:"version"`
	SchemaVersion        int    `json:"schema_version"`
	DatabasePath         string `json:"database_path"`
	ActiveMemories       int    `json:"active_memories"`
	UserScopeActiveCount int    `json:"user_scope_active_count"`
	EmbeddingCount       int    `json:"embedding_count"`
	EntityLinkCount      int    `json:"entity_link_count"`
	DefaultUserID        string `json:"default_user_id"`
	DefaultAgentID       string `json:"default_agent_id"`
	DefaultRunID         string `json:"default_run_id"`
	HybridEnabled        bool   `json:"hybrid_enabled"`
}

// Doctor collects diagnostics for the open store and current environment defaults.
func (s *Store) Doctor(ctx context.Context) (*DoctorReport, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	active, err := s.Stats(ctx)
	if err != nil {
		return nil, err
	}
	userID := ResolveUserID()
	var userScope int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM memories WHERE valid_to = ? AND user_id = ?`, "", userID,
	).Scan(&userScope); err != nil {
		return nil, fmt.Errorf("count user scope: %w", err)
	}
	var embedCount, entityCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM memory_embeddings`).Scan(&embedCount); err != nil {
		return nil, fmt.Errorf("count embeddings: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM memory_entities`).Scan(&entityCount); err != nil {
		return nil, fmt.Errorf("count entities: %w", err)
	}
	return &DoctorReport{
		Version:              Version,
		SchemaVersion:        SchemaVersion,
		DatabasePath:         s.Path(),
		ActiveMemories:       active,
		UserScopeActiveCount: userScope,
		EmbeddingCount:       embedCount,
		EntityLinkCount:      entityCount,
		DefaultUserID:        userID,
		DefaultAgentID:       ResolveAgentID(),
		DefaultRunID:         ResolveRunID(),
		HybridEnabled:        HybridEnabled(),
	}, nil
}

// FormatDoctorReport renders a human-readable doctor summary for the CLI.
func FormatDoctorReport(r *DoctorReport) string {
	if r == nil {
		return "memex doctor: no report"
	}
	hybrid := "disabled"
	if r.HybridEnabled {
		hybrid = "enabled"
	}
	return fmt.Sprintf(`memex doctor (%s)
schema:      %d
database:    %s
active:      %d memories (%d in user_id=%q)
hybrid:      %s (%d embeddings indexed)
entities:    %d indexed links
defaults:    user_id=%q agent_id=%q run_id=%q
status:      ok
`, r.Version, r.SchemaVersion, r.DatabasePath, r.ActiveMemories, r.UserScopeActiveCount, r.DefaultUserID,
		hybrid, r.EmbeddingCount, r.EntityLinkCount, r.DefaultUserID, r.DefaultAgentID, r.DefaultRunID)
}
