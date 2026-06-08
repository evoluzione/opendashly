package storage

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// WithQueryMemory returns a context carrying per-query ClickHouse memory
// settings that override the connection-level defaults. The adaptive tuning
// controller uses this to raise or lower the dashboard query budget at runtime
// without rebuilding the client. A non-positive budget leaves ctx untouched so
// the connection defaults apply.
func WithQueryMemory(ctx context.Context, maxMemoryBytes int) context.Context {
	if maxMemoryBytes <= 0 {
		return ctx
	}
	spill := maxMemoryBytes / 4
	return clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		"max_memory_usage":                   maxMemoryBytes,
		"max_bytes_before_external_group_by": spill,
		"max_bytes_before_external_sort":     spill,
	}))
}
