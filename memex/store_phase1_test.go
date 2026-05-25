package memex

import (
	"strings"
	"testing"
)

func TestGetForgetScopedByAgentID(t *testing.T) {
	store, ctx := openTestStore(t)

	agentMem, err := store.Remember(ctx, "agent secret", nil, "note",
		WithUserID("alice"), WithAgentID("agent-a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "other agent", nil, "note",
		WithUserID("alice"), WithAgentID("agent-b")); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(ctx, agentMem.ID, "alice", "agent-b"); err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("wrong agent get = %v", err)
	}
	if err := store.Forget(ctx, agentMem.ID, "alice", "agent-b"); err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("wrong agent forget = %v", err)
	}

	got, err := store.Get(ctx, agentMem.ID, "alice", "agent-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentID != "agent-a" {
		t.Fatalf("agent_id = %q", got.AgentID)
	}
}

func TestListFiltersByAgentRunAndMetadata(t *testing.T) {
	store, ctx := openTestStore(t)

	if _, err := store.Remember(ctx, "run one", nil, "note",
		WithUserID("u1"), WithAgentID("a1"), WithRunID("r1"),
		WithMetadata(map[string]any{"source": "agent", "tier": "gold"})); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "run two", nil, "note",
		WithUserID("u1"), WithAgentID("a1"), WithRunID("r2"),
		WithMetadata(map[string]any{"source": "user"})); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "other agent", nil, "note",
		WithUserID("u1"), WithAgentID("a2"), WithRunID("r1")); err != nil {
		t.Fatal(err)
	}

	byRun, err := store.List(ctx, MemoryFilter{UserID: "u1", AgentID: "a1", RunID: "r1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(byRun) != 1 || byRun[0].Content != "run one" {
		t.Fatalf("run filter = %+v", byRun)
	}

	byMeta, err := store.List(ctx, MemoryFilter{
		UserID: "u1", Metadata: map[string]string{"source": "agent"}, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byMeta) != 1 || byMeta[0].Content != "run one" {
		t.Fatalf("metadata filter = %+v", byMeta)
	}
}

func TestUpdateScopedByUserID(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "owned by alice", nil, "note", WithUserID("alice"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Update(ctx, mem.ID, "hijacked", nil, "", nil, "bob", "")
	if err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("cross-user update = %v", err)
	}

	updated, err := store.Update(ctx, mem.ID, "revised", nil, "", nil, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "revised" {
		t.Fatalf("update = %+v", updated)
	}
}

func TestHistoryScopedByUserID(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "v1", nil, "note", WithUserID("alice"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(ctx, mem.ID, "v2", nil, "", nil, "alice", ""); err != nil {
		t.Fatal(err)
	}

	if _, err := store.History(ctx, mem.ID, "bob"); err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("cross-user history = %v", err)
	}
	hist, err := store.History(ctx, mem.ID, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) < 2 {
		t.Fatalf("history = %+v", hist)
	}
}

func TestResolveAgentAndRunIDFromEnv(t *testing.T) {
	t.Setenv("MEMEX_AGENT_ID", "env-agent")
	t.Setenv("MEMEX_RUN_ID", "env-run")
	if got := ResolveAgentIDArg(""); got != "env-agent" {
		t.Fatalf("ResolveAgentIDArg = %q", got)
	}
	if got := ResolveRunIDArg("explicit"); got != "explicit" {
		t.Fatalf("ResolveRunIDArg = %q", got)
	}
}

func TestMetadataFilterRejectsInvalidKey(t *testing.T) {
	store, ctx := openTestStore(t)
	_, err := store.List(ctx, MemoryFilter{Metadata: map[string]string{"bad-key": "x"}, Limit: 5})
	if err == nil || !strings.Contains(err.Error(), "invalid metadata filter key") {
		t.Fatalf("List() error = %v", err)
	}
}
