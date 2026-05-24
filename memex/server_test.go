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
