package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestBuildLatencyDistributionQuery_UsesNormalizedSpanKindFilter(t *testing.T) {
	query := BuildLatencyDistributionQuery(time.Unix(0, 0), time.Unix(3600, 0), "")

	if !strings.Contains(query, "upper(toString(SpanKind)) IN ('SERVER', 'SPAN_KIND_SERVER')") {
		t.Fatalf("expected normalized SpanKind filter, got: %s", query)
	}
	if !strings.Contains(query, "toString(SpanKind) = '2'") {
		t.Fatalf("expected numeric SpanKind fallback, got: %s", query)
	}
}

func TestBuildStatusCodeBreakdownQuery_UsesNormalizedStatusCodeFilter(t *testing.T) {
	query := BuildStatusCodeBreakdownQuery(time.Unix(0, 0), time.Unix(3600, 0), "")

	if !strings.Contains(query, "toString(StatusCode) IN ('Ok', 'STATUS_CODE_OK', '1')") {
		t.Fatalf("expected normalized ok status filter, got: %s", query)
	}
	if !strings.Contains(query, "toString(StatusCode) IN ('Error', 'STATUS_CODE_ERROR', '2')") {
		t.Fatalf("expected normalized error status filter, got: %s", query)
	}
}
