package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"opendashly/backend/internal/api"
	"opendashly/backend/internal/query"
)

func TestSavedQueriesContract(t *testing.T) {
	queryService := &query.Service{}
	relatedService := &query.RelatedService{}
	savedRepo := query.NewSavedQueryRepo()
	handler := api.NewRouter(api.RouterConfig{
		QueryService:   queryService,
		RelatedService: relatedService,
		SavedRepo:      savedRepo,
	})

	payload := map[string]any{
		"name":        "Errors",
		"description": "",
		"request": map[string]any{
			"signals": []string{"logs"},
			"timeRange": map[string]string{
				"from": "2026-01-01T00:00:00Z",
				"to":   "2026-01-02T00:00:00Z",
			},
			"filters": map[string]string{"service.name": "api"},
		},
	}
	body, _ := json.Marshal(payload)

	createReq := httptest.NewRequest(http.MethodPost, "/api/queries", bytes.NewReader(body))
	createReq.Header.Set("X-Tenant-ID", "t1")
	createReq.Header.Set("X-User-ID", "u1")
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRec.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/queries", nil)
	listReq.Header.Set("X-Tenant-ID", "t1")
	listReq.Header.Set("X-User-ID", "u1")
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRec.Code)
	}
}
