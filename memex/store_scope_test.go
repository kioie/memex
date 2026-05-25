package memex

import (
	"strings"
	"testing"
)

func TestGetForgetScopedByUserID(t *testing.T) {
	store, ctx := openTestStore(t)

	aliceMem, err := store.Remember(ctx, "alice secret", nil, "note", WithUserID("alice"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Remember(ctx, "bob secret", nil, "note", WithUserID("bob")); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(ctx, aliceMem.ID, "bob", ""); err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("bob reading alice memory = %v", err)
	}
	if err := store.Forget(ctx, aliceMem.ID, "bob", ""); err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("bob deleting alice memory = %v", err)
	}

	got, err := store.Get(ctx, aliceMem.ID, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "alice secret" {
		t.Fatalf("alice get = %+v", got)
	}
	if err := store.Forget(ctx, aliceMem.ID, "alice", ""); err != nil {
		t.Fatal(err)
	}
	gotAfter, err := store.Get(ctx, aliceMem.ID, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotAfter.ValidTo == nil {
		t.Fatal("expected soft-deleted memory")
	}
	active, err := store.List(ctx, MemoryFilter{UserID: "alice", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatal("expected no active memories after scoped forget")
	}
}

func TestRememberRejectsOversizedContent(t *testing.T) {
	store, ctx := openTestStore(t)
	tooLong := strings.Repeat("x", maxMemoryContentLen+1)
	_, err := store.Remember(ctx, tooLong, nil, "note")
	if err == nil || !strings.Contains(err.Error(), "maximum length") {
		t.Fatalf("Remember() error = %v", err)
	}
}

func TestUpdateRejectsOversizedContent(t *testing.T) {
	store, ctx := openTestStore(t)
	mem, err := store.Remember(ctx, "small", nil, "note")
	if err != nil {
		t.Fatal(err)
	}
	tooLong := strings.Repeat("y", maxMemoryContentLen+1)
	_, err = store.Update(ctx, mem.ID, UpdateInput{Content: tooLong})
	if err == nil || !strings.Contains(err.Error(), "maximum length") {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestMCPGetForgetRespectUserScope(t *testing.T) {
	store, ctx := openTestStore(t)
	h := &toolHandlers{store: store}

	rememberRes, _, err := h.remember(ctx, nil, rememberArgs{Content: "scoped", UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	id := extractRememberedID(t, textFromResult(rememberRes))

	_, _, err = h.getMemory(ctx, nil, getArgs{ID: id, UserID: "bob"})
	if err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("cross-user get_memory = %v", err)
	}
	_, _, err = h.forget(ctx, nil, forgetArgs{ID: id, UserID: "bob"})
	if err == nil || !strings.Contains(err.Error(), "memory not found") {
		t.Fatalf("cross-user forget = %v", err)
	}

	getRes, _, err := h.getMemory(ctx, nil, getArgs{ID: id, UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(textFromResult(getRes), "scoped") {
		t.Fatalf("scoped get = %q", textFromResult(getRes))
	}
}
