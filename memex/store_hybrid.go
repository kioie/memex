package memex

import (
	"fmt"
)

const hybridCandidateLimit = 50

func (s *Store) hybridSearch(query string, filter MemoryFilter, limit, offset int) ([]Memory, error) {
	userID := ResolveUserIDArg(filter.UserID)
	candidateFilter := filter
	candidateFilter.Limit = hybridCandidateLimit
	candidateFilter.Offset = 0

	ftsResults, err := s.searchFTS(query, candidateFilter, hybridCandidateLimit, 0)
	if err != nil {
		return nil, err
	}

	queryEntities := extractEntities(query, nil)
	entityResults, err := s.searchEntities(userID, queryEntities, candidateFilter, hybridCandidateLimit)
	if err != nil {
		return nil, err
	}

	vectorResults, err := s.searchVector(userID, query, candidateFilter, hybridCandidateLimit)
	if err != nil {
		return nil, err
	}

	lists := [][]string{
		memoryIDsFromMemories(ftsResults),
		memoryIDsFromMemories(entityResults),
		memoryIDsFromMemories(vectorResults),
	}
	signalCount := 0
	for _, list := range lists {
		if len(list) > 0 {
			signalCount++
		}
	}
	if signalCount <= 1 {
		switch {
		case len(ftsResults) > 0:
			return paginateMemories(ftsResults, limit, offset), nil
		case len(entityResults) > 0:
			return paginateMemories(entityResults, limit, offset), nil
		default:
			return paginateMemories(vectorResults, limit, offset), nil
		}
	}

	fusedIDs := reciprocalRankFusion(lists)
	scores := mergeRRFScores(lists)
	memories, err := s.fetchFilteredMemoriesByIDs(fusedIDs, filter)
	if err != nil {
		return nil, err
	}
	order := indexOfIDs(fusedIDs)
	sortMemoriesByIDOrder(&memories, order, scores)
	return paginateMemories(memories, limit, offset), nil
}

func (s *Store) searchFTS(query string, filter MemoryFilter, limit, offset int) ([]Memory, error) {
	ftsQuery := buildFTSQuery(query)
	sqlText, args, err := searchMemoriesSQL(ftsQuery, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()
	return scanMemoriesSearch(rows)
}

func paginateMemories(memories []Memory, limit, offset int) []Memory {
	if offset >= len(memories) {
		return nil
	}
	memories = memories[offset:]
	if limit > 0 && len(memories) > limit {
		memories = memories[:limit]
	}
	return memories
}

func indexOfIDs(ids []string) map[string]int {
	order := make(map[string]int, len(ids))
	for i, id := range ids {
		order[id] = i
	}
	return order
}

func sortMemoriesByIDOrder(memories *[]Memory, order map[string]int, scores map[string]float64) {
	list := *memories
	for i := 1; i < len(list); i++ {
		j := i
		for j > 0 && order[list[j].ID] < order[list[j-1].ID] {
			list[j], list[j-1] = list[j-1], list[j]
			j--
		}
	}
	for i := range list {
		if score, ok := scores[list[i].ID]; ok {
			list[i].Score = score
		}
	}
	*memories = list
}
