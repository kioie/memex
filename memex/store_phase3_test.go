package memex

import (
	"strings"
	"testing"
)

func TestRememberAgentFactTypesDefaultSourceAgent(t *testing.T) {
	store, ctx := openTestStore(t)
	for _, memoryType := range []string{"commitment", "recommendation", "action_taken"} {
		mem, err := store.Remember(ctx, memoryType+" fact", nil, memoryType)
		if err != nil {
			t.Fatalf("Remember(%q): %v", memoryType, err)
		}
		if mem.Source != SourceAgent {
			t.Fatalf("type %q source = %q, want agent", memoryType, mem.Source)
		}
	}
}

func TestRememberExplicitSourceAndValidation(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "system note", nil, "note", WithSource(SourceSystem))
	if err != nil {
		t.Fatal(err)
	}
	if mem.Source != SourceSystem {
		t.Fatalf("source = %q", mem.Source)
	}
	_, err = store.Remember(ctx, "bad", nil, "note", WithSource("llm"))
	if err == nil || !strings.Contains(err.Error(), "invalid source") {
		t.Fatalf("Remember invalid source error = %v", err)
	}
	_, err = store.Remember(ctx, "bad type", nil, "unknown")
	if err == nil || !strings.Contains(err.Error(), "invalid memory type") {
		t.Fatalf("Remember invalid type error = %v", err)
	}
}

func TestRecallFiltersBySource(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "user prefers tea", nil, "preference", WithSource(SourceUser)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "will follow up tomorrow", nil, "commitment"); err != nil {
		t.Fatal(err)
	}
	agentOnly, err := store.Search(ctx, "", MemoryFilter{Source: SourceAgent, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(agentOnly) != 1 || agentOnly[0].Type != "commitment" {
		t.Fatalf("agent filter = %+v", agentOnly)
	}
	userOnly, err := store.List(ctx, MemoryFilter{Source: SourceUser, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(userOnly) != 1 || userOnly[0].Type != "preference" {
		t.Fatalf("user filter = %+v", userOnly)
	}
}

func TestUpdateSupersedePreservesSourceUnlessOverridden(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "ship v1", nil, "commitment")
	if err != nil {
		t.Fatal(err)
	}
	if mem.Source != SourceAgent {
		t.Fatalf("source = %q", mem.Source)
	}
	updated, err := store.Update(ctx, mem.ID, UpdateInput{Content: "ship v2"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Source != SourceAgent {
		t.Fatalf("supersede preserved source = %q", updated.Source)
	}
	withSource, err := store.Update(ctx, updated.ID, UpdateInput{Content: "ship v3", Source: SourceSystem})
	if err != nil {
		t.Fatal(err)
	}
	if withSource.Source != SourceSystem {
		t.Fatalf("supersede override source = %q", withSource.Source)
	}
}

func TestFilterClausesSQLSourceUsesBoundParameter(t *testing.T) {
	args := []any{"user-1"}
	sql, err := filterClausesSQL(MemoryFilter{Source: SourceAgent}, &args, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, clauseFilterSource) {
		t.Fatalf("filter SQL = %q", sql)
	}
	if len(args) != 3 || args[1] != SourceAgent {
		t.Fatalf("args = %v", args)
	}
}
