package builders

import (
	"strings"
	"testing"
	"time"
)

// StatusCode is stored as 'Error', '2' or 'STATUS_CODE_ERROR' depending on the
// OTel ingestion path. The traces list must flag a trace as failed for any of
// these, matching the span detail view and the rest of the codebase.
func TestBuildTracesQuery_ErrorCountMatchesAllStatusRepresentations(t *testing.T) {
	from := time.Unix(0, 0)
	to := time.Unix(3600, 0)

	query := BuildTracesQuery(nil, nil, from, to, 50, 0, nil)

	if !strings.Contains(query, "toString(StatusCode) IN ('Error', '2', 'STATUS_CODE_ERROR')") {
		t.Fatalf("expected errorCount to match all status representations, got: %s", query)
	}
	if strings.Contains(query, "StatusCode = 'STATUS_CODE_ERROR'") {
		t.Fatalf("expected query to drop narrow STATUS_CODE_ERROR-only match, got: %s", query)
	}
}

func TestBuildTracesCountQuery_ErrorCountMatchesAllStatusRepresentations(t *testing.T) {
	from := time.Unix(0, 0)
	to := time.Unix(3600, 0)

	query := BuildTracesCountQuery(nil, nil, from, to)

	if !strings.Contains(query, "toString(StatusCode) IN ('Error', '2', 'STATUS_CODE_ERROR')") {
		t.Fatalf("expected errorCount to match all status representations, got: %s", query)
	}
	if strings.Contains(query, "StatusCode = 'STATUS_CODE_ERROR'") {
		t.Fatalf("expected query to drop narrow STATUS_CODE_ERROR-only match, got: %s", query)
	}
}
