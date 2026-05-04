package query

import (
	"testing"
	"time"
)

func TestNormalizeRunPagination_DefaultsAndOffsets(t *testing.T) {
	p := normalizeRunPagination(QueryRequest{Page: 0, Limit: 0})
	if p.page != 1 || p.limit != 100 {
		t.Fatalf("unexpected defaults: page=%d limit=%d", p.page, p.limit)
	}
	if p.offset != 0 || p.readLimit != 101 {
		t.Fatalf("unexpected defaults offset/readLimit: offset=%d readLimit=%d", p.offset, p.readLimit)
	}

	p = normalizeRunPagination(QueryRequest{Page: 3, Limit: 25})
	if p.page != 3 || p.limit != 25 || p.offset != 50 || p.readLimit != 26 {
		t.Fatalf("unexpected explicit pagination: %#v", p)
	}
}

func TestBuildSignalQueries_InvalidCursorReturnsError(t *testing.T) {
	_, err := buildSignalQueries(QueryRequest{LogsCursor: "%%%"}, runPagination{page: 1, limit: 10, offset: 0, readLimit: 11})
	if err == nil {
		t.Fatal("expected error for invalid logs cursor")
	}

	_, err = buildSignalQueries(QueryRequest{TracesCursor: "%%%"}, runPagination{page: 1, limit: 10, offset: 0, readLimit: 11})
	if err == nil {
		t.Fatal("expected error for invalid traces cursor")
	}
}

func TestAssembleQueryRunResult_PopulatesSummaryPaginationAndErrors(t *testing.T) {
	p := runPagination{page: 2, limit: 5, offset: 5, readLimit: 6}
	result := assembleQueryRunResult(p, "partial", signalExecutionResult{
		logs:           []LogEntry{{Timestamp: time.Now()}},
		traces:         []TraceEntry{{TraceID: "tr-1"}, {TraceID: "tr-2"}},
		logsHasNext:    true,
		logsNextCursor: "abc",
		tracesHasNext:  false,
		signalErrors:   map[string]string{"traces": "timeout"},
	})

	if result.Status != "partial" {
		t.Fatalf("unexpected status %q", result.Status)
	}
	if result.Summary.LogCount != 1 || result.Summary.TraceCount != 2 {
		t.Fatalf("unexpected summary %#v", result.Summary)
	}
	if !result.Pagination.Logs.HasNext || result.Pagination.Logs.NextCursor != "abc" {
		t.Fatalf("unexpected logs pagination %#v", result.Pagination.Logs)
	}
	if result.Pagination.Traces.NextCursor != "" {
		t.Fatalf("expected empty traces cursor when hasNext is false, got %q", result.Pagination.Traces.NextCursor)
	}
	if result.SignalErrors["traces"] != "timeout" {
		t.Fatalf("unexpected signal errors %#v", result.SignalErrors)
	}
}
