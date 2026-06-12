package storage

import (
	"context"
	"sync"
	"testing"
)

// TestQueryMemorySettings_ConcurrentCallersGetIndependentMaps reproduces the
// production "fatal error: concurrent map writes" crash that killed the backend.
//
// clickhouse-go writes max_execution_time into a query's settings map on every
// call (when the context has a deadline). The dashboard fans out many queries
// concurrently over one shared context via errgroup, so if those queries shared
// a single settings map, the concurrent writes raced and the Go runtime aborted
// the whole process. queryMemorySettings must therefore hand each query its own
// map.
//
// The sustained write loop makes Go's built-in "concurrent map writes" detector
// fire deterministically even without the race detector: with a shared map this
// crashes the test process; with a fresh map per call it passes.
func TestQueryMemorySettings_ConcurrentCallersGetIndependentMaps(t *testing.T) {
	ctx := WithQueryMemory(context.Background(), 64<<20)

	const (
		goroutines = 64
		writes     = 2000
	)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			settings, ok := queryMemorySettings(ctx)
			if !ok {
				t.Error("expected query memory settings to be present")
				return
			}
			// Mirror clickhouse-go's queryOptions, which writes
			// max_execution_time into the per-query settings map.
			for j := 0; j < writes; j++ {
				settings["max_execution_time"] = j
			}
		}()
	}
	wg.Wait()
}

// TestWithQueryMemory_NoBudgetLeavesContextUntouched documents that a
// non-positive budget is a no-op so connection-level defaults apply.
func TestWithQueryMemory_NoBudgetLeavesContextUntouched(t *testing.T) {
	ctx := context.Background()
	if got := WithQueryMemory(ctx, 0); got != ctx {
		t.Fatal("non-positive budget must return the original context unchanged")
	}
	if _, ok := queryMemorySettings(ctx); ok {
		t.Fatal("expected no settings without a budget")
	}
}

// TestQueryMemorySettings_AppliesBudgetAndSpill pins the derived values.
func TestQueryMemorySettings_AppliesBudgetAndSpill(t *testing.T) {
	const budget = 64 << 20
	ctx := WithQueryMemory(context.Background(), budget)
	settings, ok := queryMemorySettings(ctx)
	if !ok {
		t.Fatal("expected settings for a positive budget")
	}
	if settings["max_memory_usage"] != budget {
		t.Fatalf("max_memory_usage = %v, want %d", settings["max_memory_usage"], budget)
	}
	if settings["max_bytes_before_external_group_by"] != budget/4 {
		t.Fatalf("spill = %v, want %d", settings["max_bytes_before_external_group_by"], budget/4)
	}
}
