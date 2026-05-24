package memex

import "strings"

// Static SQL fragments for optional memory filters. User-supplied values are always
// passed as bound parameters (?), never interpolated into the query text.
const (
	sqlSelectMemories = `
		SELECT id, content, tags, memory_type, created_at, updated_at, metadata, user_id
		FROM memories WHERE user_id = ?`
	sqlSelectMemoriesSearch = `
		SELECT m.id, m.content, m.tags, m.memory_type, m.created_at, m.updated_at, bm25(memories_fts) AS score,
		       snippet(memories_fts, 0, '[', ']', '…', 12) AS highlights, m.metadata, m.user_id
		FROM memories_fts
		JOIN memories m ON m.rowid = memories_fts.rowid
		WHERE memories_fts MATCH ? AND m.user_id = ?`
	clauseFilterMemoryType = ` AND memory_type = ?`
	clauseFilterTag          = ` AND tags LIKE ?`
	suffixOrderUpdated       = ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	suffixOrderScore         = ` ORDER BY score LIMIT ? OFFSET ?`
)

func listMemoriesSQL(filter MemoryFilter, limit, offset int) (string, []any) {
	userID := ResolveUserIDArg(filter.UserID)
	args := []any{userID}
	filterSQL := filterClausesSQL(filter, &args)
	return sqlSelectMemories + filterSQL + suffixOrderUpdated, append(args, limit, offset)
}

func searchMemoriesSQL(ftsQuery string, filter MemoryFilter, limit, offset int) (string, []any) {
	userID := ResolveUserIDArg(filter.UserID)
	args := []any{ftsQuery, userID}
	filterSQL := filterClausesSQL(filter, &args)
	return sqlSelectMemoriesSearch + filterSQL + suffixOrderScore, append(args, limit, offset)
}

func filterClausesSQL(filter MemoryFilter, args *[]any) string {
	var clauses strings.Builder
	if t := strings.TrimSpace(filter.Type); t != "" {
		clauses.WriteString(clauseFilterMemoryType)
		*args = append(*args, t)
	}
	for _, tag := range normalizeTags(filter.Tags) {
		clauses.WriteString(clauseFilterTag)
		*args = append(*args, tagLikePattern(tag))
	}
	return clauses.String()
}

func tagLikePattern(tag string) string {
	tag = strings.NewReplacer(`%`, "", `_`, "", `"`, "", `\`, "").Replace(tag)
	return `%"` + tag + `"%`
}
