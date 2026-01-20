package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opendashly/backend/internal/api"
	"opendashly/backend/internal/api/handlers"
	"opendashly/backend/internal/auth"
	"opendashly/backend/internal/query"
)

func TestServiceFilterFlowIntegration(t *testing.T) {
	repo := newMemoryRepo()
	hash, _ := auth.HashPassword("admin")
	_, _ = repo.Create(context.Background(), auth.User{
		ID:                 "u-admin",
		Username:           "admin",
		PasswordHash:       hash,
		Role:               auth.RoleAdmin,
		MustChangePassword: false,
	})

	secret := []byte("test-secret")
	authHandler := &handlers.AuthHandler{Repo: repo, Secret: secret, CookieName: "session", SessionTTL: time.Hour}
	servicesHandler := &handlers.ServicesHandler{Service: &query.Service{}}
	middleware := auth.Middleware(auth.MiddlewareOptions{
		Mode:           "jwt",
		CookieName:     "session",
		JWTSecret:      secret,
		Repo:           repo,
		TenantID:       "default",
		AllowlistPaths: []string{"/api/auth/login"},
	})

	router := api.NewRouter(api.RouterConfig{
		QueryService:    &query.Service{},
		RelatedService:  &query.RelatedService{},
		SavedRepo:       query.NewSavedQueryRepo(),
		AuthHandler:     authHandler,
		ServicesHandler: servicesHandler,
		AuthMiddleware:  middleware,
	})

	loginPayload, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginPayload))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d", loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()

	servicesReq := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	servicesReq.AddCookie(cookies[0])
	servicesRec := httptest.NewRecorder()
	router.ServeHTTP(servicesRec, servicesReq)
	if servicesRec.Code != http.StatusOK {
		t.Fatalf("expected services 200, got %d", servicesRec.Code)
	}
}
