package memex

import (
	"fmt"
	"strings"
	"testing"
)

func TestRetrieveContextPacksWithinBudget(t *testing.T) {
	store, ctx := openTestStore(t)
	for i := range 20 {
		content := fmt.Sprintf("%s keyword-pack-%02d", strings.Repeat("x", 200), i)
		if _, err := store.Remember(ctx, content, nil, "note"); err != nil {
			t.Fatal(err)
		}
	}

	result, err := store.RetrieveContext(ctx, "keyword-pack", MemoryFilter{}, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Memories) == 0 {
		t.Fatal("expected at least one packed memory")
	}
	if result.TokenCount > result.MaxTokens {
		t.Fatalf("token_count %d exceeds max_tokens %d", result.TokenCount, result.MaxTokens)
	}
	if !result.Truncated {
		t.Fatal("expected truncation with small token budget")
	}
}

func TestRetrieveContextRequiresQuery(t *testing.T) {
	store, ctx := openTestStore(t)
	_, err := store.RetrieveContext(ctx, "  ", MemoryFilter{}, 1000)
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("RetrieveContext error = %v", err)
	}
}

func TestRecallHandlerRequiresQuery(t *testing.T) {
	store, ctx := openTestStore(t)
	h := &toolHandlers{store: store}
	_, _, err := h.recall(ctx, nil, recallArgs{})
	if err == nil || !strings.Contains(err.Error(), "list_memories") {
		t.Fatalf("recall empty query error = %v", err)
	}
}

func TestRetrieveContextHandler(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "bounded context retrieval", nil, "fact"); err != nil {
		t.Fatal(err)
	}
	h := &toolHandlers{store: store}
	res, _, err := h.retrieveContext(ctx, nil, retrieveContextArgs{
		Query:     "bounded",
		MaxTokens: 4096,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := textFromResult(res)
	if !strings.Contains(text, "bounded context") || !strings.Contains(text, "token_count") {
		t.Fatalf("retrieve_context = %q", text)
	}
}
