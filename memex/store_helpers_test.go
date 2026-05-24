package memex

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func openTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, t.Context()
}

func seedMemories(t *testing.T, store *Store, ctx context.Context, n int, prefix string) []string {
	t.Helper()
	ids := make([]string, 0, n)
	for i := range n {
		content := fmt.Sprintf("%s entry %04d keyword-%s", prefix, i, parityKeyword(i))
		mem, err := store.Remember(ctx, content, []string{prefix, parityKeyword(i)}, "note")
		if err != nil {
			t.Fatalf("remember #%d: %v", i, err)
		}
		ids = append(ids, mem.ID)
	}
	return ids
}

func parityKeyword(i int) string {
	if i%2 == 0 {
		return "even"
	}
	return "odd"
}

func stringsRepeat(s string, n int) string {
	return strings.Repeat(s, n)
}
