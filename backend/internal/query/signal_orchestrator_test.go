package query

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func TestSignalOrchestrator_RunSuccessWithPaginationAndCursor(t *testing.T) {
	orchestrator := newSignalOrchestrator(
		func(context.Context, driver.Conn, string) ([]LogEntry, error) {
			return []LogEntry{
				{Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TraceID: "t1", SpanID: "s1"},
				{Timestamp: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC), TraceID: "t2", SpanID: "s2"},
			}, nil
		},
		func(context.Context, driver.Conn, string) ([]TraceEntry, error) {
			return []TraceEntry{
				{TraceID: "tr-1", LastSeen: time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC)},
			}, nil
		},
		func(context.Context, driver.Conn, string) ([]MetricSeries, error) {
			return []MetricSeries{{Name: "m1"}, {Name: "m2"}}, nil
		},
	)

	result := orchestrator.run(context.Background(), nil, signalQueries{}, map[string]bool{"logs": true, "traces": true, "metrics": true}, 1)

	if len(result.signalErrors) != 0 {
		t.Fatalf("expected no signal errors, got %#v", result.signalErrors)
	}
	if len(result.logs) != 1 || !result.logsHasNext {
		t.Fatalf("expected logs trimmed to 1 with hasNext=true, got len=%d hasNext=%v", len(result.logs), result.logsHasNext)
	}
	if result.logsNextCursor == "" {
		t.Fatal("expected non-empty logs cursor")
	}
	if len(result.traces) != 1 || result.tracesHasNext {
		t.Fatalf("expected traces len=1 and hasNext=false, got len=%d hasNext=%v", len(result.traces), result.tracesHasNext)
	}
	if len(result.metrics) != 1 || !result.metricsHasNext {
		t.Fatalf("expected metrics trimmed to 1 with hasNext=true, got len=%d hasNext=%v", len(result.metrics), result.metricsHasNext)
	}
}

func TestSignalOrchestrator_RunCollectsPartialErrors(t *testing.T) {
	orchestrator := newSignalOrchestrator(
		func(context.Context, driver.Conn, string) ([]LogEntry, error) {
			return nil, errors.New("logs unavailable")
		},
		func(context.Context, driver.Conn, string) ([]TraceEntry, error) {
			return []TraceEntry{{TraceID: "ok"}}, nil
		},
		func(context.Context, driver.Conn, string) ([]MetricSeries, error) {
			return nil, errors.New("metrics unavailable")
		},
	)

	result := orchestrator.run(context.Background(), nil, signalQueries{}, map[string]bool{"logs": true, "traces": true, "metrics": true}, 100)

	if got := result.signalErrors["logs"]; got != "logs unavailable" {
		t.Fatalf("unexpected logs error: %q", got)
	}
	if got := result.signalErrors["metrics"]; got != "metrics unavailable" {
		t.Fatalf("unexpected metrics error: %q", got)
	}
	if len(result.traces) != 1 {
		t.Fatalf("expected traces to be returned, got len=%d", len(result.traces))
	}
}
