package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter_HealthzReturns200(t *testing.T) {
	h := NewRouter(RouterConfig{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestNewRouter_CORSPreflightReturns204AndHeaders(t *testing.T) {
	h := NewRouter(RouterConfig{CORSAllowedOrigins: []string{"http://localhost:5173"}})
	req := httptest.NewRequest(http.MethodOptions, "/api/query/run", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin to be echoed, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Access-Control-Allow-Methods header")
	}
}

func TestNewRouter_UsesProvidedAuthMiddleware(t *testing.T) {
	called := false
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	h := NewRouter(RouterConfig{AuthMiddleware: middleware})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected auth middleware to be called")
	}
}
