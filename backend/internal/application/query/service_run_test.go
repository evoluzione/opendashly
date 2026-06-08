package query

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"opendashly/backend/internal/application/pressure"
	"opendashly/backend/internal/infrastructure/storage"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type fakeSignalRunner struct {
	called bool
	result signalExecutionResult
}

func (f *fakeSignalRunner) run(context.Context, driver.Conn, signalQueries, map[string]bool, int) signalExecutionResult {
	f.called = true
	return f.result
}

func TestRun_UsesInjectedSignalRunnerAndReturnsPartial(t *testing.T) {
	fake := &fakeSignalRunner{
		result: signalExecutionResult{
			logs:         []LogEntry{{Timestamp: time.Now(), Body: "ok"}},
			signalErrors: map[string]string{"traces": "timeout"},
		},
	}

	svc := &Service{Storage: &storage.Client{}, signalRunner: fake}
	res, err := svc.Run(context.Background(), QueryRequest{
		Signals:   []string{"logs", "traces"},
		TimeRange: TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
		Page:      1,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fake.called {
		t.Fatal("expected injected signal runner to be called")
	}
	if res.Status != "partial" {
		t.Fatalf("expected partial status, got %q", res.Status)
	}
	if got := res.SignalErrors["traces"]; got != "timeout" {
		t.Fatalf("unexpected signal error: %q", got)
	}
	if res.Summary.LogCount != 1 {
		t.Fatalf("expected log count 1, got %d", res.Summary.LogCount)
	}
}

func TestRun_AllRequestedSignalsFailedReturnsError(t *testing.T) {
	fake := &fakeSignalRunner{
		result: signalExecutionResult{
			signalErrors: map[string]string{
				"logs":   "downstream failure",
				"traces": "timeout",
			},
		},
	}

	svc := &Service{Storage: &storage.Client{}, signalRunner: fake}
	_, err := svc.Run(context.Background(), QueryRequest{
		Signals:   []string{"logs", "traces"},
		TimeRange: TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
		Page:      1,
		Limit:     10,
	})
	if err == nil {
		t.Fatal("expected error when all requested signals fail")
	}
	if !strings.Contains(err.Error(), "all requested signals failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_StorageNilReturnsEmptyResultWithDefaults(t *testing.T) {
	svc := &Service{}

	res, err := svc.Run(context.Background(), QueryRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "complete" {
		t.Fatalf("expected complete status, got %q", res.Status)
	}
	if res.Pagination.Logs.Page != 1 || res.Pagination.Logs.Limit != 100 {
		t.Fatalf("unexpected default pagination: %#v", res.Pagination.Logs)
	}
	if res.Summary.LogCount != 0 || res.Summary.TraceCount != 0 {
		t.Fatalf("expected empty summary, got %#v", res.Summary)
	}
}

func TestRun_ReturnsBusyWithoutStartingSignalQueriesWhenLimiterIsSaturated(t *testing.T) {
	limiter := pressure.NewLimiter(1)
	release, ok := limiter.TryAcquire()
	if !ok {
		t.Fatal("expected limiter setup acquire to succeed")
	}
	defer release()

	fake := &fakeSignalRunner{}
	svc := &Service{
		Storage:      &storage.Client{},
		Limiter:      limiter,
		signalRunner: fake,
	}

	_, err := svc.Run(context.Background(), QueryRequest{
		Signals:   []string{"logs"},
		TimeRange: TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
	})

	if !errors.Is(err, pressure.ErrBusy) {
		t.Fatalf("expected ErrBusy, got %v", err)
	}
	if fake.called {
		t.Fatal("signal runner should not start when limiter is saturated")
	}
}

func TestRun_InvalidLogsCursorReturnsError(t *testing.T) {
	svc := &Service{Storage: &storage.Client{}}

	_, err := svc.Run(context.Background(), QueryRequest{
		Signals:    []string{"logs"},
		LogsCursor: "%%%",
		TimeRange:  TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
	})
	if err == nil {
		t.Fatal("expected error for invalid logs cursor")
	}
	if !strings.Contains(err.Error(), "decode logs cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_InvalidTracesCursorReturnsError(t *testing.T) {
	svc := &Service{Storage: &storage.Client{}}

	_, err := svc.Run(context.Background(), QueryRequest{
		Signals:      []string{"traces"},
		TracesCursor: "%%%",
		TimeRange:    TimeRange{From: time.Now().Add(-time.Hour), To: time.Now()},
	})
	if err == nil {
		t.Fatal("expected error for invalid traces cursor")
	}
	if !strings.Contains(err.Error(), "decode traces cursor") {
		t.Fatalf("unexpected error: %v", err)
	}
}
