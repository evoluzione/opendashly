package query

import "testing"

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
