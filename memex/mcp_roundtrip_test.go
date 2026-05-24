package memex

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP roundtrip tests attach via server.RawServer() — tinymcp exposes the underlying go-sdk server.

func TestMCPRememberRecallForgetRoundtrip(t *testing.T) {
	store, ctx := openTestStore(t)
	server, err := NewMCPServer(store)
	if err != nil {
		t.Fatal(err)
	}

	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.RawServer().Connect(ctx, t1, nil); err != nil {
		t.Fatal(err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 9 {
		t.Fatalf("expected 9 tools, got %d", len(tools.Tools))
	}

	rememberRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remember",
		Arguments: map[string]any{
			"content": "Prefers table-driven tests in Go",
			"tags":    []string{"testing", "go"},
			"type":    "preference",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rememberText := textFromResult(rememberRes)
	if !strings.Contains(rememberText, "Remembered [") {
		t.Fatalf("unexpected remember response: %q", rememberText)
	}

	recallRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "recall",
		Arguments: map[string]any{
			"query": "table-driven",
			"limit": 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	recallText := textFromResult(recallRes)
	if !strings.Contains(recallText, "table-driven") {
		t.Fatalf("unexpected recall response: %q", recallText)
	}

	id := extractRememberedID(t, rememberText)
	getRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_memory",
		Arguments: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(getRes), id) {
		t.Fatalf("get_memory missing id %q", id)
	}

	forgetRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "forget",
		Arguments: map[string]any{
			"id": id,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(forgetRes), "Forgot memory") {
		t.Fatalf("unexpected forget response: %q", textFromResult(forgetRes))
	}

	recallAfter, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "recall",
		Arguments: map[string]any{
			"query": "table-driven",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(recallAfter), "No memories matched") {
		t.Fatalf("expected empty recall after forget, got %q", textFromResult(recallAfter))
	}
}

func TestMCPRememberValidationError(t *testing.T) {
	store, ctx := openTestStore(t)
	server, err := NewMCPServer(store)
	if err != nil {
		t.Fatal(err)
	}

	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.RawServer().Connect(ctx, t1, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "remember",
		Arguments: map[string]any{"content": "   "},
	})
	if err != nil {
		return // protocol-level error is also acceptable
	}
	if res == nil || !res.IsError {
		t.Fatal("expected validation error for empty content")
	}
	if !strings.Contains(textFromResult(res), "content is required") {
		t.Fatalf("unexpected error text: %q", textFromResult(res))
	}
}

func extractRememberedID(t *testing.T, rememberText string) string {
	t.Helper()
	start := strings.Index(rememberText, "[")
	end := strings.Index(rememberText, "]")
	if start < 0 || end <= start {
		t.Fatalf("could not parse id from %q", rememberText)
	}
	return rememberText[start+1 : end]
}
