package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opendashly/backend/internal/query"
)

func TestQueryHandler_BadJSONReturns400(t *testing.T) {
	h := &QueryHandler{Service: &query.Service{}}
	req := httptest.NewRequest(http.MethodPost, "/api/query/run", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestQueryHandler_InvalidCursorReturns200WhenStorageMissing(t *testing.T) {
	h := &QueryHandler{Service: &query.Service{}}
	payload := map[string]any{
		"signals":    []string{"logs"},
		"timeRange":  map[string]string{"from": "2026-01-01T00:00:00Z", "to": "2026-01-02T00:00:00Z"},
		"logsCursor": "not-base64",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/query/run", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestQueryHandler_SuccessReturnsJSON(t *testing.T) {
	h := &QueryHandler{Service: &query.Service{}}
	payload := map[string]any{
		"signals":   []string{"logs"},
		"timeRange": map[string]string{"from": "2026-01-01T00:00:00Z", "to": "2026-01-02T00:00:00Z"},
		"page":      1,
		"limit":     25,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/query/run", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", got)
	}

	var result query.QueryRunResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("expected valid json response, got error: %v", err)
	}
	if result.Status != "complete" {
		t.Fatalf("expected complete status, got %q", result.Status)
	}
}
