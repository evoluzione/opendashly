package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opendashly/backend/internal/query"
)

func TestAttributesHandler_ReturnsEmptyListWhenStorageMissing(t *testing.T) {
	h := &AttributesHandler{Service: &query.Service{}}
	req := httptest.NewRequest(http.MethodGet, "/api/query/attributes?q=service", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", got)
	}

	var keys []string
	if err := json.NewDecoder(rec.Body).Decode(&keys); err != nil {
		t.Fatalf("expected valid json array, got error: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected empty keys, got %v", keys)
	}
}
