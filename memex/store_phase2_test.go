package memex

import (
	"strings"
	"testing"
)

func TestUpdateSupersedesAndHidesOldFromRecall(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "v1", []string{"x"}, "fact")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(ctx, mem.ID, UpdateInput{
		Content:  "v2",
		Tags:     []string{"y"},
		Type:     "decision",
		Metadata: map[string]any{"source": "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID == mem.ID {
		t.Fatalf("expected new id, got same %q", updated.ID)
	}
	if updated.SupersedesID != mem.ID {
		t.Fatalf("supersedes_id = %q, want %q", updated.SupersedesID, mem.ID)
	}

	old, err := store.Get(ctx, mem.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if old.ValidTo == nil {
		t.Fatal("superseded row should have valid_to set")
	}

	active, err := store.List(ctx, MemoryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ID != updated.ID {
		t.Fatalf("active list = %+v", active)
	}

	found, err := store.Search(ctx, "v1", MemoryFilter{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("superseded content should not appear in search: %+v", found)
	}
}

func TestForgetSoftDeleteExcludesFromRecall(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "delete me", nil, "note")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Forget(ctx, mem.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, mem.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ValidTo == nil {
		t.Fatal("expected soft-deleted memory to have valid_to")
	}
	list, err := store.List(ctx, MemoryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("active list = %+v", list)
	}
	inactive, err := store.List(ctx, MemoryFilter{Limit: 10, IncludeInactive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(inactive) != 1 {
		t.Fatalf("inactive list = %+v", inactive)
	}
}

func TestMemoryEventsRecordedOnAddSupersedeDelete(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "event test", nil, "note")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(ctx, mem.ID, UpdateInput{Content: "event test v2"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Forget(ctx, updated.ID, "", ""); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM memory_events WHERE memory_id IN (?, ?)`, mem.ID, updated.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count < 3 {
		t.Fatalf("memory_events count = %d", count)
	}
}

func TestUpdateRejectsSupersededID(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "once", nil, "note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(ctx, mem.ID, UpdateInput{Content: "twice"}); err != nil {
		t.Fatal(err)
	}
	_, err = store.Update(ctx, mem.ID, UpdateInput{Content: "thrice"})
	if err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("update superseded id = %v", err)
	}
}
