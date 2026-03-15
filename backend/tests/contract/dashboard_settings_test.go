package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/dashboard"
	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/interfaces/api"
	"opendashly/backend/internal/interfaces/http/handlers"
)

type contractDashboardRepo struct{}

func (r *contractDashboardRepo) GetSettings(context.Context, string) ([]dashboard.ChartSetting, error) {
	return nil, nil
}

func (r *contractDashboardRepo) UpsertSettings(context.Context, string, []dashboard.ChartSetting, string) error {
	return nil
}

func TestDashboardSettingsGetContract(t *testing.T) {
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
	settingsSvc := &dashboard.Service{Repo: &contractDashboardRepo{}}
	middleware := auth.Middleware(auth.MiddlewareOptions{
		Mode:           "jwt",
		CookieName:     "session",
		JWTSecret:      secret,
		Repo:           repo,
		TenantID:       "default",
		AllowlistPaths: []string{"/api/auth/login"},
	})

	router := api.NewRouter(api.RouterConfig{
		QueryService:      &query.Service{},
		RelatedService:    &query.RelatedService{},
		SavedRepo:         query.NewSavedQueryRepo(),
		AuthHandler:       authHandler,
		DashboardSettings: settingsSvc,
		AuthMiddleware:    middleware,
	})

	loginPayload, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginPayload))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d", loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected login cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/settings", nil)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var res dashboard.SettingsResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(res.Settings) == 0 {
		t.Fatalf("expected default settings, got none")
	}

	first := res.Settings[0]
	if first.Key == "" || first.W <= 0 || first.H <= 0 {
		t.Fatalf("expected layout fields in response, got %+v", first)
	}
}
