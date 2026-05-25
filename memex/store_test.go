package memex

import (
	"path/filepath"
	"testing"
)

func TestStoreRememberRecallForget(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := t.Context()
	mem, err := store.Remember(ctx, "Prefers table-driven tests in Go", []string{"testing", "go"}, "preference")
	if err != nil {
		t.Fatal(err)
	}
	if mem.ID == "" {
		t.Fatal("expected id")
	}

	got, err := store.Get(ctx, mem.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != mem.Content {
		t.Fatalf("content mismatch: %q", got.Content)
	}

	results, err := store.Recall(ctx, "table-driven", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != mem.ID {
		t.Fatalf("expected id %s, got %s", mem.ID, results[0].ID)
	}

	if err := store.Forget(ctx, mem.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, mem.ID, ""); err == nil {
		t.Fatal("expected error after forget")
	}
}

func TestStoreListRecent(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := t.Context()
	for _, content := range []string{"alpha", "beta", "gamma"} {
		if _, err := store.Remember(ctx, content, nil, "note"); err != nil {
			t.Fatal(err)
		}
	}

	results, err := store.Recall(ctx, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestOpenCreatesDatabaseFile(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	want := filepath.Join(dir, "memex.db")
	if store.Path() != want {
		t.Fatalf("path = %q, want %q", store.Path(), want)
	}
}
