package memex

import (
	"fmt"
	"strings"
	"testing"
)

func TestRememberStoresAllFields(t *testing.T) {
	store, ctx := openTestStore(t)

	mem, err := store.Remember(ctx, "  Uses spaceship operator in Rust  ", []string{" Rust ", "rust", ""}, "preference")
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ctx, mem.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.Content != "Uses spaceship operator in Rust" {
		t.Fatalf("content = %q", got.Content)
	}
	if got.Type != "preference" {
		t.Fatalf("type = %q", got.Type)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "rust" {
		t.Fatalf("tags = %v, want [rust] (deduped, lowercased)", got.Tags)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps")
	}
}

func TestRememberDefaultType(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "plain note", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if mem.Type != "note" {
		t.Fatalf("type = %q, want note", mem.Type)
	}
}

func TestRecallSearchByContentAndTags(t *testing.T) {
	store, ctx := openTestStore(t)

	if _, err := store.Remember(ctx, "alpha project uses PostgreSQL", []string{"database"}, "fact"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "beta project uses SQLite", []string{"database"}, "fact"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "unrelated cooking recipe", []string{"food"}, "note"); err != nil {
		t.Fatal(err)
	}

	byContent, err := store.Recall(ctx, "PostgreSQL", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(byContent) != 1 || !strings.Contains(byContent[0].Content, "PostgreSQL") {
		t.Fatalf("content search = %+v", byContent)
	}

	byTag, err := store.Recall(ctx, "database", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(byTag) != 2 {
		t.Fatalf("tag search count = %d, want 2", len(byTag))
	}
}

func TestRecallNoMatches(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "hello world", nil, "note"); err != nil {
		t.Fatal(err)
	}
	results, err := store.Recall(ctx, "xyzzyplugh", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestForgetRemovesFromSearch(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "unique-token-for-deletion-test", nil, "note")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Forget(ctx, mem.ID); err != nil {
		t.Fatal(err)
	}
	results, err := store.Recall(ctx, "unique-token-for-deletion", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("deleted memory still searchable: %+v", results)
	}
}

func TestStatsReflectsStoreSize(t *testing.T) {
	store, ctx := openTestStore(t)
	count, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("initial count = %d", count)
	}

	for i := range 3 {
		if _, err := store.Remember(ctx, fmt.Sprintf("item %d", i), nil, "note"); err != nil {
			t.Fatal(err)
		}
	}
	count, err = store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
}

func TestRecallLimitDefaultsAndBounds(t *testing.T) {
	store, ctx := openTestStore(t)
	seedMemories(t, store, ctx, 15, "limit")

	tests := []struct {
		name      string
		limit     int
		wantCount int
	}{
		{name: "zero defaults to 10", limit: 0, wantCount: 10},
		{name: "negative defaults to 10", limit: -1, wantCount: 10},
		{name: "explicit 5", limit: 5, wantCount: 5},
		{name: "explicit 15", limit: 15, wantCount: 15},
		{name: "over total returns all", limit: 100, wantCount: 15},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := store.Recall(ctx, "", tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != tc.wantCount {
				t.Fatalf("got %d results, want %d", len(results), tc.wantCount)
			}
		})
	}
}

func TestLargeContentStorage(t *testing.T) {
	sizes := []int{1 << 10, 64 << 10} // 1 KiB, 64 KiB
	if !testing.Short() {
		sizes = append(sizes, 1<<20) // 1 MiB when not -short
	}

	for _, size := range sizes {
		t.Run(formatSize(size), func(t *testing.T) {
			store, ctx := openTestStore(t)
			token := fmt.Sprintf("hello-%d", size)
			payload := token + " " + stringsRepeat("x", size-len(token)-1)
			mem, err := store.Remember(ctx, payload, []string{"large"}, "fact")
			if err != nil {
				t.Fatalf("Remember %d bytes: %v", size, err)
			}
			got, err := store.Get(ctx, mem.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Content) != size {
				t.Fatalf("stored len = %d, want %d", len(got.Content), size)
			}
			results, err := store.Recall(ctx, token, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 1 || results[0].ID != mem.ID {
				t.Fatalf("FTS did not find large memory: %+v", results)
			}
		})
	}
}

func formatSize(n int) string {
	switch {
	case n >= 1<<20:
		return "1MiB"
	case n >= 64<<10:
		return "64KiB"
	default:
		return "1KiB"
	}
}

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "hello world", want: `"hello" OR "world"`},
		{in: `say "hello"`, want: `"say" OR "hello"`},
		{in: "  spaced   tokens  ", want: `"spaced" OR "tokens"`},
	}

	for _, tc := range tests {
		if got := buildFTSQuery(tc.in); got != tc.want {
			t.Fatalf("buildFTSQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeTags(t *testing.T) {
	got := normalizeTags([]string{" Go ", "GO", "", "rust", "Rust"})
	if len(got) != 2 || got[0] != "go" || got[1] != "rust" {
		t.Fatalf("normalizeTags = %v", got)
	}
}
