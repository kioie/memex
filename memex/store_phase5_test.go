package memex

import (
	"strings"
	"testing"
	"time"
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

func TestHybridSearchRespectsAgentFilter(t *testing.T) {
	t.Setenv("MEMEX_HYBRID", "1")
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "shared alpha beta gamma topic", nil, "note", WithAgentID("agent-a")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "shared alpha beta gamma topic", nil, "note", WithAgentID("agent-b")); err != nil {
		t.Fatal(err)
	}
	results, err := store.Search(ctx, "alpha beta gamma", MemoryFilter{AgentID: "agent-a", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].AgentID != "agent-a" {
		t.Fatalf("filtered hybrid search = %+v", results)
	}
}

func TestReciprocalRankFusionOrdersSharedHits(t *testing.T) {
	fused := reciprocalRankFusion([][]string{
		{"a", "b", "c"},
		{"b", "a", "d"},
	})
	if len(fused) != 4 {
		t.Fatalf("RRF order = %v", fused)
	}
	top := map[string]struct{}{fused[0]: {}, fused[1]: {}}
	_, hasA := top["a"]
	_, hasB := top["b"]
	if !hasA || !hasB {
		t.Fatalf("expected a and b in top two, got %v", fused)
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

func TestMemoryMatchesFilter(t *testing.T) {
	now := time.Now().UTC()
	validTo := now
	mem := &Memory{
		Type:     "fact",
		AgentID:  "agent-1",
		RunID:    "run-1",
		Source:   SourceAgent,
		Tags:     []string{"billing"},
		Metadata: map[string]any{"env": "prod"},
	}
	if !memoryMatchesFilter(mem, MemoryFilter{}) {
		t.Fatal("empty filter should match")
	}
	if memoryMatchesFilter(mem, MemoryFilter{Type: "note"}) {
		t.Fatal("type mismatch should not match")
	}
	if memoryMatchesFilter(mem, MemoryFilter{AgentID: "other"}) {
		t.Fatal("agent mismatch should not match")
	}
	if memoryMatchesFilter(mem, MemoryFilter{RunID: "run-2"}) {
		t.Fatal("run mismatch should not match")
	}
	if memoryMatchesFilter(mem, MemoryFilter{Source: SourceUser}) {
		t.Fatal("source mismatch should not match")
	}
	if memoryMatchesFilter(mem, MemoryFilter{Tags: []string{"missing"}}) {
		t.Fatal("tag mismatch should not match")
	}
	if memoryMatchesFilter(mem, MemoryFilter{Metadata: map[string]string{"env": "dev"}}) {
		t.Fatal("metadata mismatch should not match")
	}
	mem.ValidTo = &validTo
	if memoryMatchesFilter(mem, MemoryFilter{}) {
		t.Fatal("inactive memory should not match default filter")
	}
	if !memoryMatchesFilter(mem, MemoryFilter{IncludeInactive: true}) {
		t.Fatal("inactive memory should match when include_inactive set")
	}
}

func TestMemoryHasTag(t *testing.T) {
	if !memoryHasTag([]string{"billing", "acme"}, "acme") {
		t.Fatal("expected tag match")
	}
	if memoryHasTag([]string{"billing"}, "acme") {
		t.Fatal("expected no tag match")
	}
}

func TestFetchFilteredMemoriesByIDsSkipsFilteredOut(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "scoped content", nil, "note", WithAgentID("keep-me"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.fetchFilteredMemoriesByIDs([]string{mem.ID}, MemoryFilter{AgentID: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected filtered out, got %+v", got)
	}
}

func TestBackfillMemoryEntitiesIndexesExistingRows(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, `Legacy Acme "Billing" row`, []string{"acme"}, "note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`DELETE FROM memory_entities WHERE memory_id = ?`, mem.ID); err != nil {
		t.Fatal(err)
	}
	if err := backfillMemoryEntities(store.db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM memory_entities WHERE memory_id = ?`, mem.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected backfilled entities")
	}
}

func TestIndexMemoryRetrievalLockedWithoutHybrid(t *testing.T) {
	t.Setenv("MEMEX_HYBRID", "0")
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "Entity index path", []string{"entitytag"}, "note")
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM memory_entities WHERE memory_id = ?`, mem.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected entities indexed on remember")
	}
	var embedCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM memory_embeddings WHERE memory_id = ?`, mem.ID).Scan(&embedCount); err != nil {
		t.Fatal(err)
	}
	if embedCount != 0 {
		t.Fatalf("expected no embedding without hybrid, got %d", embedCount)
	}
}

func TestSupersedeReindexesEntities(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "Original Acme fact", []string{"acme"}, "fact")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(ctx, mem.ID, UpdateInput{Content: "Updated Globex platform rollout", Tags: []string{"globex"}})
	if err != nil {
		t.Fatal(err)
	}
	var globexCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM memory_entities WHERE memory_id = ? AND entity = 'globex'`, updated.ID).Scan(&globexCount); err != nil {
		t.Fatal(err)
	}
	if globexCount == 0 {
		t.Fatal("expected globex entity after supersede")
	}
}
