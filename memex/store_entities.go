package memex

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const maxQueryEntities = 32

var (
	entityQuotedPattern = regexp.MustCompile(`"([^"]{2,64})"`)
	entityTokenPattern  = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_-]{2,}`)
)

var entityStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "are": {}, "but": {}, "not": {}, "you": {},
	"all": {}, "can": {}, "had": {}, "her": {}, "was": {}, "one": {}, "our": {},
	"out": {}, "day": {}, "get": {}, "has": {}, "him": {}, "his": {}, "how": {},
	"its": {}, "may": {}, "new": {}, "now": {}, "old": {}, "see": {}, "two": {},
	"who": {}, "boy": {}, "did": {}, "let": {}, "put": {}, "say": {}, "she": {},
	"too": {}, "use": {}, "with": {}, "from": {}, "that": {}, "this": {}, "have": {},
	"will": {}, "your": {}, "what": {}, "when": {}, "where": {}, "which": {}, "about": {},
}

// extractEntities derives searchable entity tokens from memory content and tags.
func extractEntities(content string, tags []string) []string {
	seen := make(map[string]struct{})
	add := func(raw string) {
		entity := normalizeEntity(raw)
		if entity == "" {
			return
		}
		if _, ok := entityStopwords[entity]; ok {
			return
		}
		if _, ok := seen[entity]; ok {
			return
		}
		seen[entity] = struct{}{}
	}
	for _, tag := range tags {
		add(tag)
	}
	for _, match := range entityQuotedPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			add(match[1])
		}
	}
	for _, token := range entityTokenPattern.FindAllString(content, -1) {
		add(token)
	}
	for field := range strings.FieldsSeq(content) {
		field = strings.Trim(field, ".,;:!?()[]{}'\"")
		if len(field) >= 3 && !isAllLower(field) {
			add(field)
		}
	}
	out := make([]string, 0, len(seen))
	for entity := range seen {
		out = append(out, entity)
	}
	return out
}

func normalizeEntity(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if len(raw) < 2 || len(raw) > 64 {
		return ""
	}
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isAllLower(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsLower(r) {
			return false
		}
	}
	return true
}

func memoryIDsFromMemories(memories []Memory) []string {
	ids := make([]string, len(memories))
	for i, mem := range memories {
		ids[i] = mem.ID
	}
	return ids
}

func mergeRRFScores(lists [][]string) map[string]float64 {
	scores := make(map[string]float64)
	for _, list := range lists {
		for rank, id := range list {
			scores[id] += 1.0 / float64(rrfK+rank+1)
		}
	}
	return scores
}

const sqlEntitySearchBase = `
		SELECT m.id, m.content, m.tags, m.memory_type, m.created_at, m.updated_at,
		       COUNT(e.entity) AS score, '' AS highlights,
		       m.metadata, m.user_id, m.agent_id, m.run_id, m.supersedes_id, m.valid_to, m.source
		FROM memory_entities e
		JOIN memories m ON m.id = e.memory_id
		WHERE e.user_id = ? AND m.user_id = ? AND e.entity IN (SELECT value FROM json_each(?))`

func (s *Store) searchEntities(userID string, entities []string, filter MemoryFilter, limit int) ([]Memory, error) {
	if len(entities) == 0 {
		return nil, nil
	}
	if len(entities) > maxQueryEntities {
		entities = entities[:maxQueryEntities]
	}
	entitiesJSON, err := json.Marshal(entities)
	if err != nil {
		return nil, fmt.Errorf("encode entity filter: %w", err)
	}
	args := []any{userID, userID, string(entitiesJSON)}
	filterSQL, err := filterClausesSQL(filter, &args, "m.")
	if err != nil {
		return nil, err
	}
	query := sqlEntitySearchBase + filterSQL + `
		GROUP BY m.id
		ORDER BY score DESC
		LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("entity search: %w", err)
	}
	defer rows.Close()
	return scanMemoriesSearch(rows)
}
