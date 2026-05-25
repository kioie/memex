package memex

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPMemoryPrompts(t *testing.T) {
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

	prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts.Prompts) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(prompts.Prompts))
	}

	guide, err := session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "memory_guide"})
	if err != nil {
		t.Fatal(err)
	}
	guideText := promptText(guide)
	for _, want := range []string{"retrieve_context", "update_memory", "When NOT to remember"} {
		if !strings.Contains(guideText, want) {
			t.Fatalf("memory_guide missing %q", want)
		}
	}

	sessionStart, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "session_start",
		Arguments: map[string]string{"run_id": "run-42"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(promptText(sessionStart), `run_id="run-42"`) {
		t.Fatalf("session_start missing run_id: %q", promptText(sessionStart))
	}

	rememberFact, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "remember_fact",
		Arguments: map[string]string{"draft": "User likes Go and table-driven tests"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(promptText(rememberFact), "User likes Go") {
		t.Fatalf("remember_fact missing draft: %q", promptText(rememberFact))
	}
}

func TestDoctorReport(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "Doctor test memory", nil, ""); err != nil {
		t.Fatal(err)
	}
	report, err := store.Doctor(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", report.SchemaVersion, SchemaVersion)
	}
	if report.ActiveMemories != 1 || report.UserScopeActiveCount != 1 {
		t.Fatalf("counts = %+v, want 1 active in default scope", report)
	}
	if !strings.Contains(FormatDoctorReport(report), "status:      ok") {
		t.Fatalf("formatted report missing ok status: %s", FormatDoctorReport(report))
	}
}

func promptText(res *mcp.GetPromptResult) string {
	if res == nil || len(res.Messages) == 0 {
		return ""
	}
	if tc, ok := res.Messages[0].Content.(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
