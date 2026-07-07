package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opendashly/backend/internal/application/auth"
)

type fakeAuthRepo struct {
	user           auth.User
	updatePassword bool
}

func (r *fakeAuthRepo) Create(context.Context, auth.User) (*auth.User, error) { return nil, nil }
func (r *fakeAuthRepo) GetByUsername(context.Context, string) (*auth.User, error) {
	return &r.user, nil
}
func (r *fakeAuthRepo) GetByID(context.Context, string) (*auth.User, error) { return &r.user, nil }
func (r *fakeAuthRepo) List(context.Context) ([]auth.User, error)           { return []auth.User{r.user}, nil }
func (r *fakeAuthRepo) UpdatePassword(context.Context, string, string, bool) error {
	r.updatePassword = true
	return nil
}
func (r *fakeAuthRepo) UpdateLastLogin(context.Context, string, time.Time) error { return nil }

func TestFirstLoginChangePasswordRejectsSamePassword(t *testing.T) {
	hash, err := auth.HashPassword("admin")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo := &fakeAuthRepo{user: auth.User{
		ID:                 "admin-id",
		Username:           "admin",
		PasswordHash:       hash,
		Role:               auth.RoleAdmin,
		MustChangePassword: true,
	}}
	secret := []byte("test-secret")
	token, err := auth.GenerateToken(secret, repo.user.ID, repo.user.Role, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := (&AuthHandler{Repo: repo, Secret: secret, CookieName: "session", SessionTTL: time.Hour}).FirstLoginChangePassword
	wrapped := auth.Middleware(auth.MiddlewareOptions{
		Mode:              "jwt",
		CookieName:        "session",
		JWTSecret:         secret,
		Repo:              repo,
		UserCache:         auth.NewUserCache(time.Minute),
		TenantID:          "default",
		UserLookupTimeout: time.Second,
	})(http.HandlerFunc(handler))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/first-login-change-password", bytes.NewBufferString(`{"newPassword":"admin"}`))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if repo.updatePassword {
		t.Fatalf("UpdatePassword should not be called when new password matches current password")
	}
}
