package contract

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"opendashly/backend/internal/api"
	"opendashly/backend/internal/query"
)

func TestTraceRelatedContract(t *testing.T) {
	queryService := &query.Service{}
	relatedService := &query.RelatedService{}
	savedRepo := query.NewSavedQueryRepo()
	handler := api.NewRouter(api.RouterConfig{
		QueryService:   queryService,
		RelatedService: relatedService,
		SavedRepo:      savedRepo,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/traces/trace-123/related", nil)
	req.Header.Set("X-Tenant-ID", "t1")
	req.Header.Set("X-User-ID", "u1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
