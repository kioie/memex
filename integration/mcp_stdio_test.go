//go:build integration

package integration_test

import (
	"context"
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
	// Exercises memex serve end-to-end: CLI → NewMCPServer → tinymcp stdio transport.
	if runtime.GOOS == "windows" {
		t.Skip("stdio subprocess integration test skipped on windows CI for now")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	dataDir := t.TempDir()
	bin := buildMemexBinary(t)

	cmd := exec.CommandContext(ctx, bin, "serve")
	cmd.Env = append(os.Environ(),
		"MEMEX_DIR="+dataDir,
		"MEMEX_VERBOSE=1",
	)

	client := mcp.NewClient(&mcp.Implementation{Name: "integration-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd, TerminateDuration: 2 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(tools.Tools))
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
