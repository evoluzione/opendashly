package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"opendashly/backend/internal/api"
	"opendashly/backend/internal/api/handlers"
	"opendashly/backend/internal/auth"
	"opendashly/backend/internal/dashboard"
	"opendashly/backend/internal/query"
)

type integrationDashboardRepo struct {
	mu       sync.Mutex
	byTenant map[string]map[string]dashboard.ChartSetting
}

func newIntegrationDashboardRepo() *integrationDashboardRepo {
	return &integrationDashboardRepo{
		byTenant: map[string]map[string]dashboard.ChartSetting{},
	}
}

func (r *integrationDashboardRepo) GetSettings(_ context.Context, tenantID string) ([]dashboard.ChartSetting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := r.byTenant[tenantID]
	if len(items) == 0 {
		return []dashboard.ChartSetting{}, nil
	}

	out := make([]dashboard.ChartSetting, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out, nil
}

func (r *integrationDashboardRepo) UpsertSettings(_ context.Context, tenantID string, settings []dashboard.ChartSetting, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byTenant[tenantID]; !ok {
		r.byTenant[tenantID] = map[string]dashboard.ChartSetting{}
	}
	for _, setting := range settings {
		r.byTenant[tenantID][setting.Key] = setting
	}
	return nil
}

func TestDashboardSettingsAdminFlowIntegration(t *testing.T) {
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
	settingsSvc := &dashboard.Service{Repo: newIntegrationDashboardRepo()}
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

	updatePayload := map[string]any{
		"settings": []map[string]any{
			{
				"key":     "apdex_gauge",
				"enabled": true,
				"order":   10,
				"x":       1,
				"y":       2,
				"w":       5,
				"h":       2,
			},
		},
	}
	body, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/admin/dashboard/settings", bytes.NewReader(body))
	updateReq.AddCookie(cookies[0])
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", updateRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/dashboard/settings", nil)
	getReq.AddCookie(cookies[0])
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected get 200, got %d", getRec.Code)
	}

	var res dashboard.SettingsResponse
	if err := json.NewDecoder(getRec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var apdex *dashboard.ChartSetting
	for i := range res.Settings {
		if res.Settings[i].Key == "apdex_gauge" {
			apdex = &res.Settings[i]
			break
		}
	}
	if apdex == nil {
		t.Fatalf("expected apdex_gauge in settings")
	}
	if apdex.X != 1 || apdex.Y != 2 || apdex.W != 5 || apdex.H != 2 {
		t.Fatalf("unexpected layout values: %+v", *apdex)
	}
}
