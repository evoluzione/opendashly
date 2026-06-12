package storage

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type queryMemoryKey struct{}

// WithQueryMemory returns a context carrying a per-query ClickHouse memory
// budget (in bytes) that overrides the connection-level defaults. The adaptive
// tuning controller uses this to raise or lower the dashboard query budget at
// runtime without rebuilding the client. A non-positive budget leaves ctx
// untouched so the connection defaults apply.
//
// The budget is stored as a plain, immutable int — NOT as a clickhouse.Context
// options object. This is deliberate: clickhouse-go mutates the options'
// settings map on every query (it writes max_execution_time when the context
// carries a deadline). A single options context shared across a concurrent
// fan-out — as the dashboard does via errgroup — therefore has multiple
// goroutines writing one shared map, which the Go runtime aborts with
// "fatal error: concurrent map writes", killing the whole process. The mutable
// settings map is instead built fresh per query in Client.Query.
func WithQueryMemory(ctx context.Context, maxMemoryBytes int) context.Context {
	if maxMemoryBytes <= 0 {
		return ctx
	}
	return context.WithValue(ctx, queryMemoryKey{}, maxMemoryBytes)
}

// queryMemorySettings returns a freshly allocated ClickHouse settings map for
// the per-query memory budget on ctx, or ok=false when no budget is set. A new
// map is returned on every call so concurrent queries never share — and never
// concurrently write — one settings map.
func queryMemorySettings(ctx context.Context) (clickhouse.Settings, bool) {
	budget, ok := ctx.Value(queryMemoryKey{}).(int)
	if !ok || budget <= 0 {
		return nil, false
	}
	spill := budget / 4
	return clickhouse.Settings{
		"max_memory_usage":                   budget,
		"max_bytes_before_external_group_by": spill,
		"max_bytes_before_external_sort":     spill,
	}, true
}
