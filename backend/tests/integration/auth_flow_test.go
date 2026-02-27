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

type memoryRepo struct {
	users  map[string]auth.User
	byName map[string]string
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{users: map[string]auth.User{}, byName: map[string]string{}}
}

func (m *memoryRepo) Create(_ context.Context, user auth.User) (*auth.User, error) {
	if user.ID == "" {
		user.ID = "u-" + user.Username
	}
	m.users[user.ID] = user
	m.byName[user.Username] = user.ID
	return &user, nil
}

func (m *memoryRepo) GetByUsername(_ context.Context, username string) (*auth.User, error) {
	id, ok := m.byName[username]
	if !ok {
		return nil, authErrNotFound{}
	}
	user := m.users[id]
	return &user, nil
}

func (m *memoryRepo) GetByID(_ context.Context, id string) (*auth.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, authErrNotFound{}
	}
	return &user, nil
}

func (m *memoryRepo) List(_ context.Context) ([]auth.User, error) {
	users := make([]auth.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users, nil
}

func (m *memoryRepo) UpdatePassword(_ context.Context, id, newHash string, mustChange bool) error {
	user := m.users[id]
	user.PasswordHash = newHash
	user.MustChangePassword = mustChange
	m.users[id] = user
	return nil
}

func (m *memoryRepo) UpdateLastLogin(_ context.Context, id string, at time.Time) error {
	user := m.users[id]
	user.LastLoginAt = &at
	m.users[id] = user
	return nil
}

type authErrNotFound struct{}

func (authErrNotFound) Error() string { return "not found" }

func TestAuthFlowIntegration(t *testing.T) {
	repo := newMemoryRepo()
	hash, _ := auth.HashPassword("admin")
	_, _ = repo.Create(context.Background(), auth.User{
		ID:                 "u-admin",
		Username:           "admin",
		PasswordHash:       hash,
		Role:               auth.RoleAdmin,
		MustChangePassword: true,
	})

	secret := []byte("test-secret")
	authHandler := &handlers.AuthHandler{Repo: repo, Secret: secret, CookieName: "session", SessionTTL: time.Hour}
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

	changePayload, _ := json.Marshal(map[string]string{"newPassword": "newpass"})
	changeReq := httptest.NewRequest(http.MethodPost, "/api/auth/first-login-change-password", bytes.NewReader(changePayload))
	changeReq.AddCookie(cookies[0])
	changeRec := httptest.NewRecorder()
	router.ServeHTTP(changeRec, changeReq)
	if changeRec.Code != http.StatusOK {
		t.Fatalf("expected change password 200, got %d", changeRec.Code)
	}
}
