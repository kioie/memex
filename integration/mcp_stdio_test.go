//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMemexStdioMCPRoundtrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdio subprocess integration test skipped on windows CI for now")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	session := connectMemex(t, ctx, t.TempDir())
	defer session.Close()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 10 {
		t.Fatalf("expected 10 tools, got %d", len(tools.Tools))
	}

	rememberRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remember",
		Arguments: map[string]any{
			"content": "Integration test memory over stdio",
			"tags":    []string{"integration"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	rememberText := textFromToolResult(rememberRes)
	if !strings.Contains(rememberText, "Remembered [") {
		t.Fatalf("unexpected remember response: %q", rememberText)
	}

	recallRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "recall",
		Arguments: map[string]any{
			"query": "stdio",
			"limit": 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromToolResult(recallRes), "stdio") {
		t.Fatalf("unexpected recall response: %q", textFromToolResult(recallRes))
	}

	prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts.Prompts) != 3 {
		t.Fatalf("expected 3 MCP prompts, got %d", len(prompts.Prompts))
	}
}

func TestMemexRetrieveContextStdio(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdio subprocess integration test skipped on windows CI for now")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	session := connectMemex(t, ctx, t.TempDir())
	defer session.Close()

	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remember",
		Arguments: map[string]any{
			"content": "Prefers retrieve_context with a tight token budget for coding agents",
			"tags":    []string{"integration", "retrieve"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "retrieve_context",
		Arguments: map[string]any{
			"query":       "token budget",
			"max_tokens":  4096,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := textFromToolResult(res)
	var payload struct {
		Memories []struct {
			Content string `json:"content"`
		} `json:"memories"`
		TokenCount int `json:"token_count"`
		MaxTokens  int `json:"max_tokens"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("parse retrieve_context JSON: %v\nbody: %s", err, text)
	}
	if len(payload.Memories) == 0 {
		t.Fatal("expected at least one memory in retrieve_context result")
	}
	if !strings.Contains(payload.Memories[0].Content, "token budget") {
		t.Fatalf("unexpected memory content: %q", payload.Memories[0].Content)
	}
	if payload.MaxTokens != 4096 {
		t.Fatalf("max_tokens = %d, want 4096", payload.MaxTokens)
	}
	if payload.TokenCount <= 0 {
		t.Fatalf("token_count = %d, want > 0", payload.TokenCount)
	}
}

func TestMemexHybridRecallStdio(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdio subprocess integration test skipped on windows CI for now")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	dataDir := t.TempDir()
	session := connectMemex(t, ctx, dataDir, "MEMEX_HYBRID=1")
	defer session.Close()

	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remember",
		Arguments: map[string]any{
			"content": "Hybrid retrieval unique phrase zephyr-quartz-integration",
			"tags":    []string{"hybrid"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "recall",
		Arguments: map[string]any{
			"query": "zephyr-quartz-integration",
			"limit": 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromToolResult(res), "zephyr-quartz-integration") {
		t.Fatalf("hybrid recall missed content: %q", textFromToolResult(res))
	}
}

func TestMemexSupersessionRecallStdio(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdio subprocess integration test skipped on windows CI for now")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	session := connectMemex(t, ctx, t.TempDir())
	defer session.Close()

	rememberRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "remember",
		Arguments: map[string]any{
			"content": "Original supersession marker alpha-tango",
			"tags":    []string{"supersession"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	oldID := extractRememberedID(t, textFromToolResult(rememberRes))

	updateRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_memory",
		Arguments: map[string]any{
			"id":      oldID,
			"content": "Revised supersession marker alpha-tango bravo",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	newID := extractRememberedID(t, textFromToolResult(updateRes))
	if newID == oldID {
		t.Fatalf("supersession should return new id, got same %q", oldID)
	}

	recallRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "recall",
		Arguments: map[string]any{
			"query": "alpha-tango",
			"limit": 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	recallText := textFromToolResult(recallRes)
	if !strings.Contains(recallText, "Revised supersession marker") {
		t.Fatalf("recall missing revised content: %q", recallText)
	}
	if strings.Contains(recallText, "Original supersession marker") {
		t.Fatalf("recall should exclude superseded row: %q", recallText)
	}
}

func connectMemex(t *testing.T, ctx context.Context, dataDir string, extraEnv ...string) *mcp.ClientSession {
	t.Helper()

	bin := buildMemexBinary(t)
	cmd := exec.CommandContext(ctx, bin, "serve")
	env := append(os.Environ(), "MEMEX_DIR="+dataDir, "MEMEX_VERBOSE=1")
	env = append(env, extraEnv...)
	cmd.Env = env

	client := mcp.NewClient(&mcp.Implementation{Name: "integration-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd, TerminateDuration: 2 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func buildMemexBinary(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "memex")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/memex")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build memex: %v\n%s", err, out)
	}
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func textFromToolResult(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
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
