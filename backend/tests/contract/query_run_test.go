package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opentelemetry-dashboard/backend/internal/api"
	"opentelemetry-dashboard/backend/internal/query"
)

func TestQueryRunContract(t *testing.T) {
	queryService := &query.Service{}
	relatedService := &query.RelatedService{}
	savedRepo := query.NewSavedQueryRepo()
	handler := api.NewRouter(queryService, relatedService, savedRepo)

	payload := map[string]any{
		"signals": []string{"logs"},
		"timeRange": map[string]string{
			"from": "2026-01-01T00:00:00Z",
			"to":   "2026-01-02T00:00:00Z",
		},
		"filters": map[string]string{"service.name": "api"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/query/run", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
