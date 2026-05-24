package memex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDirUsesEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MEMEX_DIR", dir)
	got, err := ResolveDir()
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ResolveDir() = %q, want %q", got, want)
	}
}

func TestRememberWithSpecialContent(t *testing.T) {
	store, ctx := openTestStore(t)

	payloads := []struct {
		content string
		query   string
	}{
		{content: `SELECT * FROM memories; DROP TABLE memories;--`, query: "SELECT"},
		{content: "<script>alert('xss')</script>", query: "script"},
		{content: "unicode: 日本語 🚀", query: "unicode"},
		{content: `quotes "double" and 'single'`, query: "quotes"},
		{content: "new\nline\ttab", query: "new"},
	}

	for _, tc := range payloads {
		mem, err := store.Remember(ctx, tc.content, []string{"security"}, "fact")
		if err != nil {
			t.Fatalf("Remember(%q): %v", tc.content, err)
		}
		got, err := store.Get(ctx, mem.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Content != strings.TrimSpace(tc.content) {
			t.Fatalf("stored content mismatch for %q", tc.content)
		}
		results, err := store.Recall(ctx, tc.query, 5)
		if err != nil {
			t.Fatalf("Recall after special content: %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("expected recall hit for %q using query %q", tc.content, tc.query)
		}
	}
}

func TestOpenCreatesNestedDataDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "memex-data")
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := os.Stat(filepath.Join(dir, "memex.db")); err != nil {
		t.Fatalf("expected database file: %v", err)
	}
}

func TestStatsAndPathOnLiveStore(t *testing.T) {
	store, ctx := openTestStore(t)
	if store.Path() == "" {
		t.Fatal("expected non-empty path")
	}
	if _, err := store.Stats(ctx); err != nil {
		t.Fatal(err)
	}
}
