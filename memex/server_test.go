package memex

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRecallHandlerLimitCap(t *testing.T) {
	store, ctx := openTestStore(t)
	seedMemories(t, store, ctx, 60, "cap")

	h := &toolHandlers{store: store}

	t.Run("default limit 10", func(t *testing.T) {
		res, _, err := h.recall(ctx, nil, recallArgs{})
		if err != nil {
			t.Fatal(err)
		}
		if countJSONMemories(t, textFromResult(res)) != 10 {
			t.Fatalf("expected 10 results in payload")
		}
	})

	t.Run("explicit 30", func(t *testing.T) {
		res, _, err := h.recall(ctx, nil, recallArgs{Limit: 30})
		if err != nil {
			t.Fatal(err)
		}
		if countJSONMemories(t, textFromResult(res)) != 30 {
			t.Fatalf("expected 30 results in payload")
		}
	})

	t.Run("capped at 50", func(t *testing.T) {
		res, _, err := h.recall(ctx, nil, recallArgs{Limit: 999})
		if err != nil {
			t.Fatal(err)
		}
		if countJSONMemories(t, textFromResult(res)) != 50 {
			t.Fatalf("expected MCP recall cap of 50")
		}
	})
}

func TestExtendedToolHandlers(t *testing.T) {
	store, ctx := openTestStore(t)
	h := &toolHandlers{store: store}

	rememberRes, _, err := h.remember(ctx, nil, rememberArgs{
		Content:  "Prefers dark mode",
		Tags:     []string{"ui"},
		Type:     "preference",
		UserID:   "alice",
		Metadata: map[string]any{"source": "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := extractRememberedID(t, textFromResult(rememberRes))

	listRes, _, err := h.listMemories(ctx, nil, listMemoriesArgs{UserID: "alice", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if countJSONMemories(t, textFromResult(listRes)) != 1 {
		t.Fatalf("list_memories = %q", textFromResult(listRes))
	}

	updateRes, _, err := h.updateMemory(ctx, nil, updateMemoryArgs{
		ID: id, Content: "Prefers light mode", Tags: []string{"ui"}, Type: "preference", UserID: "alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(updateRes), "Prefers light mode") {
		t.Fatalf("update_memory = %q", textFromResult(updateRes))
	}

	histRes, _, err := h.memoryHistory(ctx, nil, memoryHistoryArgs{ID: id, UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(histRes), historyUpdate) {
		t.Fatalf("memory_history = %q", textFromResult(histRes))
	}

	m2, _, err := h.remember(ctx, nil, rememberArgs{Content: "Second memory", UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	id2 := extractRememberedID(t, textFromResult(m2))

	delRes, _, err := h.deleteMemories(ctx, nil, deleteMemoriesArgs{IDs: []string{id, id2}, UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(delRes), "Deleted 2 memories") {
		t.Fatalf("delete_memories = %q", textFromResult(delRes))
	}

	m3, _, err := h.remember(ctx, nil, rememberArgs{Content: "Wipe me", UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	id3 := extractRememberedID(t, textFromResult(m3))

	_, _, err = h.deleteAllMemories(ctx, nil, deleteAllMemoriesArgs{UserID: "alice"})
	if err == nil || !strings.Contains(err.Error(), "confirm=true") {
		t.Fatalf("delete_all without confirm = %v", err)
	}

	wipeRes, _, err := h.deleteAllMemories(ctx, nil, deleteAllMemoriesArgs{UserID: "alice", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(wipeRes), "Deleted all 1 memories") {
		t.Fatalf("delete_all_memories = %q", textFromResult(wipeRes))
	}
	if id3 == "" {
		t.Fatal("setup failed")
	}
}

func TestToolHandlersPropagateStoreErrors(t *testing.T) {
	store, ctx := openTestStore(t)
	h := &toolHandlers{store: store}

	_, _, err := h.remember(ctx, nil, rememberArgs{Content: ""})
	if err == nil || !strings.Contains(err.Error(), "content is required") {
		t.Fatalf("remember error = %v", err)
	}

	_, _, err = h.getMemory(ctx, nil, getArgs{ID: "missing"})
	if err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("get_memory error = %v", err)
	}

	_, _, err = h.forget(ctx, nil, forgetArgs{ID: "missing"})
	if err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("forget error = %v", err)
	}
}

func TestFormatRecallEmptyStates(t *testing.T) {
	if got := formatRecall(nil, ""); got != "No memories stored yet." {
		t.Fatalf("empty store message = %q", got)
	}
	if got := formatRecall(nil, "missing"); got != `No memories matched "missing".` {
		t.Fatalf("no match message = %q", got)
	}
}

func TestFormatRememberedIncludesID(t *testing.T) {
	text := formatRemembered(&Memory{
		ID:      "abc-123",
		Content: "likes Go",
		Tags:    []string{"lang"},
		Type:    "preference",
	})
	if !strings.Contains(text, "abc-123") || !strings.Contains(text, "likes Go") {
		t.Fatalf("formatRemembered = %q", text)
	}
}

func textFromResult(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func countJSONMemories(t *testing.T, payload string) int {
	t.Helper()
	if strings.HasPrefix(payload, "No memories") {
		return 0
	}
	count := strings.Count(payload, `"id"`)
	return count
}
