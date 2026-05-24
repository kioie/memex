package memex

import (
	"strings"
	"testing"
)

func TestOpenErrors(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		_, err := Open("")
		if err == nil {
			t.Fatal("expected error for empty dir")
		}
		if !strings.Contains(err.Error(), "dir is required") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("whitespace dir", func(t *testing.T) {
		_, err := Open("   ")
		if err == nil {
			t.Fatal("expected error for whitespace dir")
		}
	})
}

func TestRememberValidationErrors(t *testing.T) {
	store, ctx := openTestStore(t)

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "empty content", content: "", wantErr: "content is required"},
		{name: "whitespace only", content: "   \n\t", wantErr: "content is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.Remember(ctx, tc.content, nil, "note")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestGetForgetValidationErrors(t *testing.T) {
	store, ctx := openTestStore(t)

	t.Run("get empty id", func(t *testing.T) {
		_, err := store.Get(ctx, "")
		if err == nil || !strings.Contains(err.Error(), "id is required") {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("get unknown id", func(t *testing.T) {
		_, err := store.Get(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil || !strings.Contains(err.Error(), "memory not found") {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("forget empty id", func(t *testing.T) {
		err := store.Forget(ctx, "  ")
		if err == nil || !strings.Contains(err.Error(), "id is required") {
			t.Fatalf("Forget() error = %v", err)
		}
	})

	t.Run("forget unknown id", func(t *testing.T) {
		err := store.Forget(ctx, "00000000-0000-0000-0000-000000000000")
		if err == nil || !strings.Contains(err.Error(), "memory not found") {
			t.Fatalf("Forget() error = %v", err)
		}
	})

	t.Run("double forget", func(t *testing.T) {
		mem, err := store.Remember(ctx, "temporary fact", nil, "note")
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Forget(ctx, mem.ID); err != nil {
			t.Fatal(err)
		}
		err = store.Forget(ctx, mem.ID)
		if err == nil || !strings.Contains(err.Error(), "memory not found") {
			t.Fatalf("second Forget() error = %v", err)
		}
	})
}

func TestClosedStoreErrors(t *testing.T) {
	store, ctx := openTestStore(t)
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "remember",
			run: func() error {
				_, err := store.Remember(ctx, "too late", nil, "note")
				return err
			},
		},
		{
			name: "recall",
			run: func() error {
				_, err := store.Recall(ctx, "x", 5)
				return err
			},
		},
		{
			name: "get",
			run: func() error {
				_, err := store.Get(ctx, "id")
				return err
			},
		},
		{
			name: "forget",
			run: func() error {
				return store.Forget(ctx, "id")
			},
		},
		{
			name: "stats",
			run: func() error {
				_, err := store.Stats(ctx)
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil || !strings.Contains(err.Error(), "store is closed") {
				t.Fatalf("%s error = %v", tc.name, err)
			}
		})
	}
}

func TestNilStoreErrors(t *testing.T) {
	var store *Store
	ctx := t.Context()

	if _, err := store.Remember(ctx, "x", nil, "note"); err == nil {
		t.Fatal("expected error on nil store Remember")
	}
	if _, err := store.Recall(ctx, "x", 5); err == nil {
		t.Fatal("expected error on nil store Recall")
	}
	if _, err := store.Get(ctx, "id"); err == nil {
		t.Fatal("expected error on nil store Get")
	}
	if err := store.Forget(ctx, "id"); err == nil {
		t.Fatal("expected error on nil store Forget")
	}
	if _, err := store.Stats(ctx); err == nil {
		t.Fatal("expected error on nil store Stats")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close on nil: %v", err)
	}
	if store.Path() != "" {
		t.Fatalf("Path on nil = %q, want empty", store.Path())
	}
}

func TestRecallQueryIsSanitized(t *testing.T) {
	store, ctx := openTestStore(t)
	if _, err := store.Remember(ctx, "foo bar baz", nil, "note"); err != nil {
		t.Fatal(err)
	}

	// buildFTSQuery quotes each token, so boolean operators in user input are not interpreted.
	results, err := store.Recall(ctx, "foo OR OR bar", 5)
	if err != nil {
		t.Fatalf("unexpected recall error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one match for sanitized query, got %d", len(results))
	}
}

func TestNewMCPServerErrors(t *testing.T) {
	_, err := NewMCPServer(nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
	if !strings.Contains(err.Error(), "store is required") {
		t.Fatalf("error = %v", err)
	}
}
