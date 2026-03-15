package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/interfaces/api"
)

func TestQueryRunContract(t *testing.T) {
	queryService := &query.Service{}
	relatedService := &query.RelatedService{}
	savedRepo := query.NewSavedQueryRepo()
	handler := api.NewRouter(api.RouterConfig{
		QueryService:   queryService,
		RelatedService: relatedService,
		SavedRepo:      savedRepo,
	})

	payload := map[string]any{
		"signals": []string{"logs"},
		"timeRange": map[string]string{
			"from": "2026-01-01T00:00:00Z",
			"to":   "2026-01-02T00:00:00Z",
		},
		"filters": map[string]string{"service.name": "api"},
		"page":    1,
		"limit":   50,
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

	var result query.QueryRunResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Pagination.Logs.Page != 1 {
		t.Fatalf("expected logs page 1, got %d", result.Pagination.Logs.Page)
	}
	if result.Pagination.Logs.Limit != 50 {
		t.Fatalf("expected logs limit 50, got %d", result.Pagination.Logs.Limit)
	}
}
