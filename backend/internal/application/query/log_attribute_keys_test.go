package query

import (
	"strings"
	"testing"
)

func TestBuildLogAttributeKeysQuery_WithoutSearch(t *testing.T) {
	query := buildLogAttributeKeysQuery("")
	if !strings.Contains(query, "SELECT DISTINCT key") {
		t.Fatalf("expected select distinct query, got %q", query)
	}
	if strings.Contains(query, "ILIKE") {
		t.Fatalf("did not expect ILIKE when search is empty, got %q", query)
	}
}

func TestBuildLogAttributeKeysQuery_WithSearchEscapesLiteral(t *testing.T) {
	query := buildLogAttributeKeysQuery("err'or")
	if !strings.Contains(query, "ILIKE") {
		t.Fatalf("expected ILIKE condition, got %q", query)
	}
	if !strings.Contains(query, "err\\'or") {
		t.Fatalf("expected escaped literal in query, got %q", query)
	}
}

func TestFallbackLogAttributeKeys_FiltersCaseInsensitive(t *testing.T) {
	filtered := fallbackLogAttributeKeys("HTTP.")
	if len(filtered) != 3 {
		t.Fatalf("expected 3 http keys, got %d (%#v)", len(filtered), filtered)
	}
	want := map[string]bool{"http.method": true, "http.route": true, "http.status_code": true}
	for _, key := range filtered {
		if !want[key] {
			t.Fatalf("unexpected key in filtered output: %q", key)
		}
	}
}
