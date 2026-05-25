package memex

import (
	"strings"
	"testing"
)

func TestExtractEntitiesFromContentAndTags(t *testing.T) {
	entities := extractEntities(`Acme project uses "GraphQL" API`, []string{"acme", "graphql"})
	if len(entities) < 3 {
		t.Fatalf("entities = %v", entities)
	}
	seen := make(map[string]struct{}, len(entities))
	for _, e := range entities {
		seen[e] = struct{}{}
	}
	for _, want := range []string{"acme", "graphql", "project"} {
		if _, ok := seen[want]; !ok {
			t.Fatalf("missing entity %q in %v", want, entities)
		}
	}
}

func TestEntitySearchBoostsNamedEntity(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "Unrelated note about cats", nil, "note"); err != nil {
		t.Fatal(err)
	}
	acme, err := store.Remember(ctx, "Acme Corp prefers PostgreSQL for billing", []string{"acme"}, "fact")
	if err != nil {
		t.Fatal(err)
	}

	results, err := store.Search(ctx, "Acme billing", MemoryFilter{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].ID != acme.ID {
		t.Fatalf("entity-boosted search = %+v", results)
	}
}

func TestHybridVectorSearchWhenEnabled(t *testing.T) {
	t.Setenv("MEMEX_HYBRID", "1")
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "alpha beta gamma database tuning", nil, "note"); err != nil {
		t.Fatal(err)
	}
	target, err := store.Remember(ctx, "alpha beta gamma vector retrieval example", nil, "note")
	if err != nil {
		t.Fatal(err)
	}

	results, err := store.Search(ctx, "alpha beta gamma retrieval", MemoryFilter{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].ID != target.ID {
		t.Fatalf("hybrid search = %+v", results)
	}
}

func TestReciprocalRankFusionOrdersSharedHits(t *testing.T) {
	fused := reciprocalRankFusion([][]string{
		{"a", "b", "c"},
		{"b", "a", "d"},
	})
	if len(fused) != 4 || fused[0] != "a" && fused[0] != "b" {
		t.Fatalf("RRF order = %v", fused)
	}
	if fused[0] != "a" {
		t.Fatalf("expected shared top ranks to prefer a, got %v", fused)
	}
}

func TestRetrieveContextUsesFusedRanking(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "Token budget pack Acme project", []string{"acme"}, "fact"); err != nil {
		t.Fatal(err)
	}
	result, err := store.RetrieveContext(ctx, "Acme project", MemoryFilter{}, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Memories) != 1 || !strings.Contains(result.Memories[0].Content, "Acme") {
		t.Fatalf("retrieve_context = %+v", result)
	}
}
