package memex

import (
	"strings"
)

// Static SQL fragments for optional memory filters. User-supplied values are always
// passed as bound parameters (?), never interpolated into the query text.
const (
	sqlMemoryColumns = `id, content, tags, memory_type, created_at, updated_at, metadata, user_id, agent_id, run_id, supersedes_id, valid_to`
	sqlSelectMemories = `
		SELECT ` + sqlMemoryColumns + `
		FROM memories WHERE user_id = ?`
	sqlSelectMemoryByHash = `
		SELECT ` + sqlMemoryColumns + `
		FROM memories WHERE user_id = ? AND content_hash = ? AND valid_to = ? LIMIT 1`
	sqlSelectMemoryByID = `
		SELECT ` + sqlMemoryColumns + `
		FROM memories WHERE id = ?`
	sqlSelectMemoryByIDUser = `
		SELECT ` + sqlMemoryColumns + `
		FROM memories WHERE id = ? AND user_id = ?`
	sqlSelectActiveByUser = `
		SELECT ` + sqlMemoryColumns + `
		FROM memories WHERE user_id = ? AND valid_to = ?`
	sqlSelectMemoriesSearch = `
		SELECT m.id, m.content, m.tags, m.memory_type, m.created_at, m.updated_at, bm25(memories_fts) AS score,
		       snippet(memories_fts, 0, '[', ']', '…', 12) AS highlights, m.metadata, m.user_id, m.agent_id, m.run_id, m.supersedes_id, m.valid_to
		FROM memories_fts
		JOIN memories m ON m.rowid = memories_fts.rowid
		WHERE memories_fts MATCH ? AND m.user_id = ?`
	clauseFilterMemoryType = ` AND memory_type = ?`
	clauseFilterTag          = ` AND tags LIKE ?`
	clauseFilterAgentID      = ` AND agent_id = ?`
	clauseFilterRunID        = ` AND run_id = ?`
	suffixOrderUpdated       = ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	suffixOrderScore         = ` ORDER BY score LIMIT ? OFFSET ?`
)

func listMemoriesSQL(filter MemoryFilter, limit, offset int) (string, []any, error) {
	userID := ResolveUserIDArg(filter.UserID)
	args := []any{userID}
	filterSQL, err := filterClausesSQL(filter, &args, "")
	if err != nil {
		return "", nil, err
	}
	return sqlSelectMemories + filterSQL + suffixOrderUpdated, append(args, limit, offset), nil
}

func searchMemoriesSQL(ftsQuery string, filter MemoryFilter, limit, offset int) (string, []any, error) {
	userID := ResolveUserIDArg(filter.UserID)
	args := []any{ftsQuery, userID}
	filterSQL, err := filterClausesSQL(filter, &args, "m.")
	if err != nil {
		return "", nil, err
	}
	return sqlSelectMemoriesSearch + filterSQL + suffixOrderScore, append(args, limit, offset), nil
}

func filterClausesSQL(filter MemoryFilter, args *[]any, colPrefix string) (string, error) {
	var clauses strings.Builder
	if t := strings.TrimSpace(filter.Type); t != "" {
		clauses.WriteString(clauseFilterMemoryType)
		*args = append(*args, t)
	}
	if agentID := ResolveAgentIDArg(filter.AgentID); agentID != "" {
		clauses.WriteString(clauseFilterAgentID)
		*args = append(*args, agentID)
	}
	if runID := ResolveRunIDArg(filter.RunID); runID != "" {
		clauses.WriteString(clauseFilterRunID)
		*args = append(*args, runID)
	}
	for _, tag := range normalizeTags(filter.Tags) {
		clauses.WriteString(clauseFilterTag)
		*args = append(*args, tagLikePattern(tag))
	}
	for key, val := range filter.Metadata {
		clause, err := metadataFilterClause(colPrefix, key)
		if err != nil {
			return "", err
		}
		clauses.WriteString(clause)
		*args = append(*args, val)
	}
	if !filter.IncludeInactive {
		clauses.WriteString(activeClause(colPrefix))
		*args = append(*args, "")
	}
	return clauses.String(), nil
}

func activeClause(colPrefix string) string {
	return ` AND ` + colPrefix + `valid_to = ?`
}

func tagLikePattern(tag string) string {
	tag = strings.NewReplacer(`%`, "", `_`, "", `"`, "", `\`, "").Replace(tag)
	return `%"` + tag + `"%`
}
