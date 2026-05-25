package memex

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"slices"
	"strings"
)

const embedDimensions = 128

const (
	sqlDeleteMemoryEntities = `DELETE FROM memory_entities WHERE memory_id = ?`
	sqlInsertMemoryEntity   = `INSERT OR IGNORE INTO memory_entities (memory_id, user_id, entity) VALUES (?, ?, ?)`
	sqlDeleteMemoryEmbedding = `DELETE FROM memory_embeddings WHERE memory_id = ?`
	sqlUpsertMemoryEmbedding = `
		INSERT INTO memory_embeddings (memory_id, user_id, embedding)
		VALUES (?, ?, ?)
		ON CONFLICT(memory_id) DO UPDATE SET user_id = excluded.user_id, embedding = excluded.embedding`
	sqlSelectEmbeddingsByUser = `SELECT memory_id, embedding FROM memory_embeddings WHERE user_id = ?`
)

// localEmbed builds a deterministic bag-of-words vector for local semantic similarity.
func localEmbed(text string) []float32 {
	vec := make([]float32, embedDimensions)
	for token := range strings.FieldsSeq(strings.ToLower(text)) {
		token = strings.Trim(token, ".,;:!?()[]{}'\"")
		if len(token) < 2 {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(token))
		idx := h.Sum32() % embedDimensions
		vec[idx] += 1
	}
	normalizeVector(vec)
	return vec
}

func normalizeVector(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		return
	}
	scale := float32(1.0 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= scale
	}
}

func encodeEmbedding(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeEmbedding(blob []byte) ([]float32, error) {
	if len(blob)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding length %d", len(blob))
	}
	vec := make([]float32, len(blob)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return vec, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

func (s *Store) replaceMemoryEntitiesLocked(memoryID, userID string, entities []string) error {
	if _, err := s.db.Exec(sqlDeleteMemoryEntities, memoryID); err != nil {
		return fmt.Errorf("delete memory entities: %w", err)
	}
	for _, entity := range entities {
		if _, err := s.db.Exec(sqlInsertMemoryEntity, memoryID, userID, entity); err != nil {
			return fmt.Errorf("insert memory entity: %w", err)
		}
	}
	return nil
}

func (s *Store) replaceMemoryEmbeddingLocked(memoryID, userID, content string) error {
	if !HybridEnabled() {
		return nil
	}
	blob := encodeEmbedding(localEmbed(content))
	if _, err := s.db.Exec(sqlDeleteMemoryEmbedding, memoryID); err != nil {
		return fmt.Errorf("delete memory embedding: %w", err)
	}
	if _, err := s.db.Exec(sqlUpsertMemoryEmbedding, memoryID, userID, blob); err != nil {
		return fmt.Errorf("upsert memory embedding: %w", err)
	}
	return nil
}

func (s *Store) indexMemoryRetrievalLocked(mem *Memory, tags []string) error {
	entities := extractEntities(mem.Content, tags)
	if err := s.replaceMemoryEntitiesLocked(mem.ID, mem.UserID, entities); err != nil {
		return err
	}
	return s.replaceMemoryEmbeddingLocked(mem.ID, mem.UserID, mem.Content)
}

type scoredMemoryID struct {
	id    string
	score float64
}

func (s *Store) searchVector(userID, query string, filter MemoryFilter, limit int) ([]Memory, error) {
	if !HybridEnabled() {
		return nil, nil
	}
	queryVec := localEmbed(query)
	rows, err := s.db.Query(sqlSelectEmbeddingsByUser, userID)
	if err != nil {
		return nil, fmt.Errorf("load embeddings: %w", err)
	}
	defer rows.Close()

	var ranked []scoredMemoryID
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, err
		}
		vec, err := decodeEmbedding(blob)
		if err != nil {
			return nil, err
		}
		ranked = append(ranked, scoredMemoryID{id: id, score: cosineSimilarity(queryVec, vec)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortScoredMemoryIDs(ranked)

	ids := make([]string, 0, min(limit, len(ranked)))
	scores := make(map[string]float64, limit)
	for _, item := range ranked {
		if len(ids) >= limit {
			break
		}
		ids = append(ids, item.id)
		scores[item.id] = item.score
	}
	memories, err := s.fetchFilteredMemoriesByIDs(ids, filter)
	if err != nil {
		return nil, err
	}
	for i := range memories {
		memories[i].Score = scores[memories[i].ID]
	}
	return memories, nil
}

func sortScoredMemoryIDs(ranked []scoredMemoryID) {
	for i := 1; i < len(ranked); i++ {
		j := i
		for j > 0 && ranked[j].score > ranked[j-1].score {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
			j--
		}
	}
}

func (s *Store) fetchFilteredMemoriesByIDs(ids []string, filter MemoryFilter) ([]Memory, error) {
	out := make([]Memory, 0, len(ids))
	for _, id := range ids {
		mem, err := s.getLocked(id)
		if err != nil {
			if stringsContainsMemoryNotFound(err) {
				continue
			}
			return nil, err
		}
		if !memoryMatchesFilter(mem, filter) {
			continue
		}
		out = append(out, *mem)
	}
	return out, nil
}

func memoryMatchesFilter(mem *Memory, filter MemoryFilter) bool {
	if !filter.IncludeInactive && mem.ValidTo != nil {
		return false
	}
	if t := strings.TrimSpace(filter.Type); t != "" && mem.Type != t {
		return false
	}
	if agentID := ResolveAgentIDArg(filter.AgentID); agentID != "" && mem.AgentID != agentID {
		return false
	}
	if runID := ResolveRunIDArg(filter.RunID); runID != "" && mem.RunID != runID {
		return false
	}
	if src := strings.TrimSpace(filter.Source); src != "" && mem.Source != src {
		return false
	}
	for _, tag := range normalizeTags(filter.Tags) {
		if !memoryHasTag(mem.Tags, tag) {
			return false
		}
	}
	for key, val := range filter.Metadata {
		if mem.Metadata == nil {
			return false
		}
		got, ok := mem.Metadata[key]
		if !ok || fmt.Sprint(got) != val {
			return false
		}
	}
	return true
}

func memoryHasTag(tags []string, want string) bool {
	return slices.Contains(tags, want)
}

func stringsContainsMemoryNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "memory not found")
}

// backfillMemoryEntities indexes entities for existing rows after migration.
func backfillMemoryEntities(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, content, tags, user_id FROM memories`)
	if err != nil {
		return err
	}
	type memoryRow struct {
		id, content, tagsJSON, userID string
	}
	var pending []memoryRow
	for rows.Next() {
		var row memoryRow
		if err := rows.Scan(&row.id, &row.content, &row.tagsJSON, &row.userID); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, row)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, row := range pending {
		var tags []string
		if row.tagsJSON != "" {
			if err := json.Unmarshal([]byte(row.tagsJSON), &tags); err != nil {
				return fmt.Errorf("decode tags: %w", err)
			}
		}
		userID := row.userID
		if userID == "" {
			userID = defaultUserID
		}
		entities := extractEntities(row.content, tags)
		if _, err := db.Exec(sqlDeleteMemoryEntities, row.id); err != nil {
			return err
		}
		for _, entity := range entities {
			if _, err := db.Exec(sqlInsertMemoryEntity, row.id, userID, entity); err != nil {
				return err
			}
		}
	}
	return nil
}
