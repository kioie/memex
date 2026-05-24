package memex

import (
	"path/filepath"
	"testing"
)

func TestStorePersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()

	var id string
	func() {
		store, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()

		mem, err := store.Remember(ctx, "survives restart", []string{"persist"}, "fact")
		if err != nil {
			t.Fatal(err)
		}
		id = mem.ID
		count, err := store.Stats(ctx)
		if err != nil || count != 1 {
			t.Fatalf("stats = %d, err = %v", count, err)
		}
	}()

	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store.Path() != filepath.Join(dir, "memex.db") {
		t.Fatalf("unexpected path %q", store.Path())
	}

	got, err := store.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "survives restart" {
		t.Fatalf("content = %q", got.Content)
	}

	results, err := store.Recall(ctx, "restart", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != id {
		t.Fatalf("recall after reopen = %+v", results)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()

	for range 3 {
		store, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		count, err := store.Stats(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected empty store on repeated Open, got %d", count)
		}
		_ = store.Close()
	}
}
