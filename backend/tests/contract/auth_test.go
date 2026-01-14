package contract

import (
	"bytes"
	"encoding/json"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opentelemetry-dashboard/backend/internal/api"
	"opentelemetry-dashboard/backend/internal/api/handlers"
	"opentelemetry-dashboard/backend/internal/auth"
	"opentelemetry-dashboard/backend/internal/query"
)

func TestAuthLoginContract(t *testing.T) {
	repo := newMemoryRepo()
	hash, err := auth.HashPassword("admin")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	_, _ = repo.Create(context.Background(), auth.User{
		ID:                 "u-admin",
		Username:           "admin",
		PasswordHash:       hash,
		Role:               auth.RoleAdmin,
		MustChangePassword: true,
	})

	secret := []byte("test-secret")
	authHandler := &handlers.AuthHandler{
		Repo:       repo,
		Secret:     secret,
		CookieName: "session",
		SessionTTL: time.Hour,
		TenantID:   "default",
	}
	middleware := auth.Middleware(auth.MiddlewareOptions{
		Mode:           "jwt",
		CookieName:     "session",
		JWTSecret:      secret,
		Repo:           repo,
		TenantID:       "default",
		AllowlistPaths: []string{"/api/auth/login"},
	})

	router := api.NewRouter(api.RouterConfig{
		QueryService:   &query.Service{},
		RelatedService: &query.RelatedService{},
		SavedRepo:      query.NewSavedQueryRepo(),
		AuthHandler:    authHandler,
		AuthMiddleware: middleware,
	})

	payload := map[string]string{"username": "admin", "password": "admin"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Fatalf("expected session cookie")
	}
	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["mustChangePassword"] != true {
		t.Fatalf("expected mustChangePassword true")
	}
}
