package memex

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	defaultRetrieveMaxTokens = 4096
	maxRetrieveMaxTokens     = 32768
	retrieveCandidateLimit   = 50
)

// RetrieveContextResult is ranked memories packed within a token budget.
type RetrieveContextResult struct {
	Memories   []Memory `json:"memories"`
	TokenCount int      `json:"token_count"`
	MaxTokens  int      `json:"max_tokens"`
	Truncated  bool     `json:"truncated"`
}

// RetrieveContext searches memories by query and greedily packs ranked hits within maxTokens.
func (s *Store) RetrieveContext(ctx context.Context, query string, filter MemoryFilter, maxTokens int) (*RetrieveContextResult, error) {
	if s == nil || s.db == nil {
		return nil, errStoreClosed
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if maxTokens <= 0 {
		maxTokens = defaultRetrieveMaxTokens
	}
	if maxTokens > maxRetrieveMaxTokens {
		maxTokens = maxRetrieveMaxTokens
	}

	filter.Limit = retrieveCandidateLimit
	filter.Offset = 0
	candidates, err := s.Search(ctx, query, filter)
	if err != nil {
		return nil, err
	}
	packed, tokenCount, truncated := packMemoriesGreedy(candidates, maxTokens)
	return &RetrieveContextResult{
		Memories:   packed,
		TokenCount: tokenCount,
		MaxTokens:  maxTokens,
		Truncated:  truncated,
	}, nil
}

func packMemoriesGreedy(candidates []Memory, maxTokens int) ([]Memory, int, bool) {
	packed := make([]Memory, 0, len(candidates))
	for i, mem := range candidates {
		trial := append(append([]Memory{}, packed...), mem)
		tokens := estimateRetrieveContextTokens(trial, maxTokens)
		if tokens > maxTokens {
			if len(packed) == 0 {
				continue
			}
			return packed, estimateRetrieveContextTokens(packed, maxTokens), i < len(candidates)
		}
		packed = trial
	}
	if len(packed) == 0 {
		return nil, 0, len(candidates) > 0
	}
	return packed, estimateRetrieveContextTokens(packed, maxTokens), len(packed) < len(candidates)
}

func estimateRetrieveContextTokens(memories []Memory, maxTokens int) int {
	payload, err := json.Marshal(&RetrieveContextResult{
		Memories:  memories,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return 0
	}
	return estimateTokensFromBytes(payload)
}

func estimateTokensFromBytes(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return (len(b) + 3) / 4
}
