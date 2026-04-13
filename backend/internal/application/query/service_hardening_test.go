package query

import (
	"strings"
	"testing"
)

func TestCountRequestedSignals(t *testing.T) {
	signals := map[string]bool{"logs": true, "traces": false, "metrics": true}
	if got := countRequestedSignals(signals); got != 2 {
		t.Fatalf("expected 2 requested signals, got %d", got)
	}
}

func TestJoinSignalErrors_SortedAndDeterministic(t *testing.T) {
	errors := map[string]string{
		"traces":  "memory limit exceeded",
		"logs":    "timeout",
		"metrics": "table missing",
	}

	got := joinSignalErrors(errors)
	want := `logs: "timeout"; metrics: "table missing"; traces: "memory limit exceeded"`
	if got != want {
		t.Fatalf("unexpected joined errors.\nwant: %s\n got: %s", want, got)
	}
}

func TestDetermineQueryRunStatus(t *testing.T) {
	tests := []struct {
		name        string
		signals     map[string]bool
		errors      map[string]string
		wantStatus  string
		wantErrLike string
	}{
		{
			name:       "complete when no errors",
			signals:    map[string]bool{"logs": true, "traces": true},
			errors:     map[string]string{},
			wantStatus: "complete",
		},
		{
			name:       "partial when at least one requested succeeds",
			signals:    map[string]bool{"logs": true, "traces": true, "metrics": false},
			errors:     map[string]string{"logs": "timeout"},
			wantStatus: "partial",
		},
		{
			name:       "partial when all requested fail with recoverable errors",
			signals:    map[string]bool{"logs": true, "traces": false, "metrics": true},
			errors:     map[string]string{"logs": "timeout", "metrics": "memory limit exceeded"},
			wantStatus: "partial",
		},
		{
			name:        "error when all requested fail with non recoverable errors",
			signals:     map[string]bool{"logs": true, "traces": false, "metrics": true},
			errors:      map[string]string{"logs": "timeout", "metrics": "down"},
			wantErrLike: "all requested signals failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, err := determineQueryRunStatus(tt.signals, tt.errors)
			if tt.wantErrLike != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrLike)
				}
				if !strings.Contains(err.Error(), tt.wantErrLike) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrLike, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotStatus != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, gotStatus)
			}
		})
	}
}
