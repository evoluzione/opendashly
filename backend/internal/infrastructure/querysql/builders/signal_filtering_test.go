package builders

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTracesQuery_IgnoresLogOnlyFilters(t *testing.T) {
	filters := []FilterItem{
		{Key: "body", Operator: "contains", Value: "timeout"},
		{Key: "severity", Operator: "=", Value: "ERROR"},
		{Key: "service.name", Operator: "=", Value: "checkout"},
	}

	query := BuildTracesQuery(nil, filters, time.Time{}, time.Time{}, 100, 0, nil)

	if strings.Contains(query, "ResourceAttributes['body']") || strings.Contains(query, "SpanAttributes['body']") {
		t.Fatalf("traces query unexpectedly contains body attribute filtering: %s", query)
	}
	if strings.Contains(strings.ToLower(query), "severity") {
		t.Fatalf("traces query unexpectedly contains severity filtering: %s", query)
	}
	if !strings.Contains(query, "ServiceName = 'checkout'") {
		t.Fatalf("traces query should keep service filter: %s", query)
	}
}

