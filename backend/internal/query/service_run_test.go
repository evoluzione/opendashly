package query

import (
	"context"
	"strings"
	"testing"
	"time"

	"opendashly/backend/internal/storage"

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
			signalErrors: map[string]string{"metrics": "timeout"},
		},
	}

	svc := &Service{Storage: &storage.Client{}, signalRunner: fake}
	res, err := svc.Run(context.Background(), QueryRequest{
		Signals:   []string{"logs", "metrics"},
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
	if got := res.SignalErrors["metrics"]; got != "timeout" {
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
				"logs":    "downstream failure",
				"metrics": "timeout",
			},
		},
	}

	svc := &Service{Storage: &storage.Client{}, signalRunner: fake}
	_, err := svc.Run(context.Background(), QueryRequest{
		Signals:   []string{"logs", "metrics"},
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
