package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type authCacheTestRepo struct {
	user  User
	err   error
	calls int
}

func (r *authCacheTestRepo) Create(context.Context, User) (*User, error) {
	return nil, errors.New("not implemented")
}

func (r *authCacheTestRepo) GetByUsername(context.Context, string) (*User, error) {
	return nil, errors.New("not implemented")
}

func (r *authCacheTestRepo) GetByID(context.Context, string) (*User, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	user := r.user
	return &user, nil
}

func (r *authCacheTestRepo) List(context.Context) ([]User, error) {
	return nil, errors.New("not implemented")
}

func (r *authCacheTestRepo) UpdatePassword(context.Context, string, string, bool) error {
	return errors.New("not implemented")
}

func (r *authCacheTestRepo) UpdateLastLogin(context.Context, string, time.Time) error {
	return errors.New("not implemented")
}

func TestMiddlewareRefreshesCachedMustChangePasswordUser(t *testing.T) {
	secret := []byte("test-secret")
	user := User{ID: "user-1", Username: "alice", Role: RoleAdmin, MustChangePassword: true}
	repo := &authCacheTestRepo{user: user}
	token, err := GenerateToken(secret, user.ID, user.Role, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	middleware := Middleware(MiddlewareOptions{
		Mode:              "jwt",
		CookieName:        "session",
		JWTSecret:         secret,
		Repo:              repo,
		TenantID:          "default",
		SessionDuration:   time.Hour,
		UserCache:         NewUserCache(time.Minute),
		UserLookupTimeout: 10 * time.Millisecond,
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	first.AddCookie(&http.Cookie{Name: "session", Value: token})
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusForbidden {
		t.Fatalf("first request status = %d, want %d", firstRec.Code, http.StatusForbidden)
	}

	repo.user.MustChangePassword = false
	second := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	second.AddCookie(&http.Cookie{Name: "session", Value: token})
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusNoContent {
		t.Fatalf("second request status = %d, want %d", secondRec.Code, http.StatusNoContent)
	}
	if repo.calls < 2 {
		t.Fatalf("expected cached must-change user to be refreshed, got %d repo calls", repo.calls)
	}
}

func TestMiddlewareUsesShortLivedUserCacheWhenDatabaseIsUnavailable(t *testing.T) {
	secret := []byte("test-secret")
	user := User{ID: "user-1", Username: "alice", Role: RoleAdmin}
	repo := &authCacheTestRepo{user: user}
	token, err := GenerateToken(secret, user.ID, user.Role, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	middleware := Middleware(MiddlewareOptions{
		Mode:              "jwt",
		CookieName:        "session",
		JWTSecret:         secret,
		Repo:              repo,
		TenantID:          "default",
		SessionDuration:   time.Hour,
		UserCache:         NewUserCache(time.Minute),
		UserLookupTimeout: 10 * time.Millisecond,
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodGet, "/api/dashboard/settings", nil)
	first.AddCookie(&http.Cookie{Name: "session", Value: token})
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want %d", firstRec.Code, http.StatusNoContent)
	}

	repo.err = errors.New("clickhouse timeout")
	second := httptest.NewRequest(http.MethodGet, "/api/dashboard/settings", nil)
	second.AddCookie(&http.Cookie{Name: "session", Value: token})
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusNoContent {
		t.Fatalf("cached request status = %d, want %d", secondRec.Code, http.StatusNoContent)
	}
	if repo.calls != 1 {
		t.Fatalf("expected cached request to avoid DB lookup, got %d repo calls", repo.calls)
	}
}
