package memex

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"
)

const (
	scaleSmoke   = 100
	scaleMedium  = 1_000
	scaleLarge   = 5_000
	scaleSearchN = 200
)

func TestScaleInsertAndStats(t *testing.T) {
	store, ctx := openTestStore(t)

	n := scaleSmoke
	if !testing.Short() {
		n = scaleMedium
	}

	ids := seedMemories(t, store, ctx, n, "scale")
	count, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != n {
		t.Fatalf("Stats() = %d, want %d", count, n)
	}

	// Every ID should be retrievable after bulk insert.
	sample := []int{0, n / 2, n - 1}
	for _, idx := range sample {
		got, err := store.Get(ctx, ids[idx], "")
		if err != nil {
			t.Fatalf("Get sample idx %d: %v", idx, err)
		}
		if got.ID != ids[idx] {
			t.Fatalf("id mismatch at %d", idx)
		}
	}
}

func TestScaleSearchPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping search scale test in -short mode")
	}

	store, ctx := openTestStore(t)
	seedMemories(t, store, ctx, scaleSearchN, "search")

	start := time.Now()
	results, err := store.Recall(ctx, "even", 25)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected matches for even keyword")
	}
	if len(results) > 25 {
		t.Fatalf("limit not respected: got %d", len(results))
	}

	// Guardrail: local FTS on 200 rows should stay well under a second.
	if elapsed > time.Second {
		t.Fatalf("recall took %v, expected < 1s for %d rows", elapsed, scaleSearchN)
	}
}

func TestScaleLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large dataset test in -short mode")
	}

	store, ctx := openTestStore(t)
	seedMemories(t, store, ctx, scaleLarge, "large")

	count, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != scaleLarge {
		t.Fatalf("count = %d, want %d", count, scaleLarge)
	}

	results, err := store.Recall(ctx, "keyword-odd", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 50 {
		t.Fatalf("limited search returned %d, want 50", len(results))
	}
	for _, mem := range results {
		if !containsSubstring(mem.Content, "keyword-odd") && !slices.Contains(mem.Tags, "odd") {
			t.Fatalf("irrelevant result: %+v", mem)
		}
	}
}

func TestConcurrentRemember(t *testing.T) {
	store, ctx := openTestStore(t)

	const workers = 8
	const perWorker = 25
	var wg sync.WaitGroup
	errs := make(chan error, workers*perWorker)

	for w := range workers {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := range perWorker {
				content := fmt.Sprintf("worker-%d item-%d", worker, i)
				if _, err := store.Remember(ctx, content, []string{"concurrent"}, "note"); err != nil {
					errs <- err
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent remember failed: %v", err)
	}

	want := workers * perWorker
	count, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("count = %d, want %d", count, want)
	}
}

func TestConcurrentRememberAndRecall(t *testing.T) {
	store, ctx := openTestStore(t)

	const writers = 6
	const readers = 4
	const perWriter = 20

	var wg sync.WaitGroup
	errs := make(chan error, writers*perWriter+readers)

	for w := range writers {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := range perWriter {
				content := fmt.Sprintf("mixed worker-%d item-%d searchable", worker, i)
				if _, err := store.Remember(ctx, content, []string{"mixed"}, "note"); err != nil {
					errs <- fmt.Errorf("remember: %w", err)
				}
			}
		}(w)
	}

	for r := range readers {
		wg.Add(1)
		go func(reader int) {
			defer wg.Done()
			for range perWriter {
				if _, err := store.Recall(ctx, "searchable", 5); err != nil {
					errs <- fmt.Errorf("recall reader %d: %w", reader, err)
				}
				if _, err := store.Stats(ctx); err != nil {
					errs <- fmt.Errorf("stats reader %d: %w", reader, err)
				}
			}
		}(r)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}

	want := writers * perWriter
	count, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("count = %d, want %d", count, want)
	}
}

func BenchmarkRemember(b *testing.B) {
	dir := b.TempDir()
	store, err := Open(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	ctx := b.Context()

	b.ResetTimer()
	for i := range b.N {
		if _, err := store.Remember(ctx, fmt.Sprintf("benchmark entry %d", i), []string{"bench"}, "note"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRecall(b *testing.B) {
	dir := b.TempDir()
	store, err := Open(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	ctx := b.Context()
	for i := range scaleSearchN {
		if _, err := store.Remember(ctx, fmt.Sprintf("bench entry %04d keyword-%s", i, parityKeyword(i)), []string{"bench"}, "note"); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for range b.N {
		if _, err := store.Recall(ctx, "even", 10); err != nil {
			b.Fatal(err)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
