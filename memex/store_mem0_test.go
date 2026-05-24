package memex

import (
	"testing"
)

func TestRememberDedupByContentHash(t *testing.T) {
	store, ctx := openTestStore(t)
	first, err := store.Remember(ctx, "Prefers Go", []string{"lang"}, "preference", WithUserID("alice"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Remember(ctx, "Prefers Go", nil, "note", WithUserID("alice"))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("dedup ids = %q vs %q", first.ID, second.ID)
	}
	other, err := store.Remember(ctx, "Prefers Go", nil, "note", WithUserID("bob"))
	if err != nil {
		t.Fatal(err)
	}
	if other.ID == first.ID {
		t.Fatal("expected different user_id scope to create new memory")
	}
}

func TestUpdateMemoryAndHistory(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "v1", []string{"x"}, "fact")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(ctx, mem.ID, "v2", []string{"y"}, "decision", map[string]any{"source": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "v2" || updated.Type != "decision" {
		t.Fatalf("update = %+v", updated)
	}
	hist, err := store.History(ctx, mem.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) < 2 {
		t.Fatalf("history = %+v", hist)
	}
	if hist[0].Action != historyUpdate {
		t.Fatalf("latest action = %q", hist[0].Action)
	}
}

func TestListAndSearchFilters(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "alpha note", []string{"a"}, "note", WithUserID("u1")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "beta decision", []string{"b"}, "decision", WithUserID("u1")); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(ctx, MemoryFilter{UserID: "u1", Type: "decision", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Type != "decision" {
		t.Fatalf("list filter = %+v", list)
	}

	found, err := store.Search(ctx, "alpha", MemoryFilter{UserID: "u1", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("search = %+v", found)
	}
}

func TestForgetBatchAndForgetAll(t *testing.T) {
	store, ctx := openTestStore(t)
	m1, _ := store.Remember(ctx, "one", nil, "note", WithUserID("wipe"))
	m2, _ := store.Remember(ctx, "two", nil, "note", WithUserID("wipe"))
	n, err := store.ForgetBatch(ctx, []string{m1.ID, m2.ID, "missing"}, "wipe")
	if err != nil || n != 2 {
		t.Fatalf("batch delete n=%d err=%v", n, err)
	}
	m3, _ := store.Remember(ctx, "three", nil, "note", WithUserID("wipe"))
	n, err = store.ForgetAll(ctx, "wipe")
	if err != nil || n != 1 {
		t.Fatalf("forget all n=%d err=%v", n, err)
	}
	if m3.ID == "" {
		t.Fatal("setup failed")
	}
}

func TestResolveUserID(t *testing.T) {
	t.Setenv("MEMEX_USER_ID", "from-env")
	if got := ResolveUserID(); got != "from-env" {
		t.Fatalf("ResolveUserID = %q", got)
	}
	t.Setenv("MEMEX_USER_ID", "")
	if got := ResolveUserID(); got != defaultUserID {
		t.Fatalf("default = %q", got)
	}
}
