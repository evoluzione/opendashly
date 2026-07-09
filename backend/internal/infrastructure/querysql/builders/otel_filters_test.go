package builders

import (
	"testing"
	"time"
)

func TestBuildOtelClauses_SeverityAllValuesDoNotFilter(t *testing.T) {
	for _, value := range []string{"", "Tutti", "All"} {
		clauses := buildOtelClauses("Timestamp", map[string]string{"severity": value}, nil, time.Time{}, time.Time{}, "ServiceName", "TraceId", "SeverityText", []string{"LogAttributes"})
		if len(clauses) != 0 {
			t.Fatalf("severity %q produced clauses: %#v", value, clauses)
		}
	}
}

func TestFilterListForSignal_TracesSkipsLogOnlyKeys(t *testing.T) {
	in := []FilterItem{
		{Key: "body", Operator: "contains", Value: "boom"},
		{Key: "severity", Operator: "=", Value: "ERROR"},
		{Key: "service.name", Operator: "=", Value: "checkout"},
	}

	out := filterListForSignal("traces", in)
	if len(out) != 1 || out[0].Key != "service.name" {
		t.Fatalf("unexpected filtered output: %#v", out)
	}
}
