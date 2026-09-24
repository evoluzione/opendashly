package builders

import (
	"strings"
	"testing"
	"time"
)

func TestErrorTracesArePrefiltered(t *testing.T) {
	from := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	to := from.Add(15 * time.Hour)
	filters := []FilterItem{
		{Key: "service.name", Operator: "=", Value: "payment-service"},
		{Connector: "AND", Key: "trace_error_scope", Operator: "=", Value: "with_errors"},
	}
	for name, q := range map[string]string{
		"page":  BuildTracesQuery(nil, filters, from, to, 100, 0, nil),
		"count": BuildTracesCountQuery(nil, filters, from, to),
	} {
		if !strings.Contains(q, "TraceId IN (SELECT TraceId FROM telemetry.otel_traces WHERE") || !strings.Contains(q, "HAVING errorCount > 0") {
			t.Errorf("%s query lacks the error prefilter: %s", name, q)
		}
	}
	// Without the errors scope nothing changes.
	if q := BuildTracesQuery(nil, filters[:1], from, to, 100, 0, nil); strings.Contains(q, "TraceId IN (") {
		t.Errorf("unexpected prefilter: %s", q)
	}
}
