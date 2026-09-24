package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/retention"
)

type retentionHandlerRepo struct {
	jobs    []retention.CleanupJob
	total   uint64
	listErr error
	countErr error
}

func (r *retentionHandlerRepo) GetSettings(context.Context) ([]retention.RetentionSetting, error) {
	return nil, nil
}

func (r *retentionHandlerRepo) GetSettingBySignal(context.Context, string) (*retention.RetentionSetting, error) {
	return nil, nil
}

func (r *retentionHandlerRepo) UpdateSetting(context.Context, string, uint32, string) error {
	return nil
}

func (r *retentionHandlerRepo) CreateCleanupJob(context.Context, retention.CleanupJob) error {
	return nil
}

func (r *retentionHandlerRepo) ListCleanupJobs(context.Context, int) ([]retention.CleanupJob, error) {
	return r.jobs, r.listErr
}

func (r *retentionHandlerRepo) CountCleanupJobs(context.Context) (uint64, error) {
	return r.total, r.countErr
}

type retentionAuthRepo struct {
	user auth.User
}

func (r *retentionAuthRepo) Create(context.Context, auth.User) (*auth.User, error) {
	return nil, nil
}

func (r *retentionAuthRepo) GetByUsername(context.Context, string) (*auth.User, error) {
	return nil, nil
}

func (r *retentionAuthRepo) GetByID(context.Context, string) (*auth.User, error) {
	return &r.user, nil
}

func (r *retentionAuthRepo) List(context.Context) ([]auth.User, error) {
	return nil, nil
}

func (r *retentionAuthRepo) UpdatePassword(context.Context, string, string, bool) error {
	return nil
}

func (r *retentionAuthRepo) UpdateLastLogin(context.Context, string, time.Time) error {
	return nil
}

func serveRetentionListJobs(t *testing.T, repo retention.Repository) *httptest.ResponseRecorder {
	t.Helper()

	secret := []byte("test-secret")
	token, err := auth.GenerateToken(secret, "admin-id", auth.RoleAdmin, time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/retention/jobs?limit=10", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	rec := httptest.NewRecorder()

	handler := &RetentionHandler{Repo: repo}
	middleware := auth.Middleware(auth.MiddlewareOptions{
		Mode:       "jwt",
		CookieName: "session",
		JWTSecret:  secret,
		Repo: &retentionAuthRepo{user: auth.User{
			ID:   "admin-id",
			Role: auth.RoleAdmin,
		}},
		TenantID: "default",
	})
	middleware(http.HandlerFunc(handler.ListJobs)).ServeHTTP(rec, req)
	return rec
}

func TestRetentionListJobs_IncludesRunningAndCompletedJobs(t *testing.T) {
	started := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	completed := started.Add(2 * time.Minute)
	repo := &retentionHandlerRepo{
		total: 2,
		jobs: []retention.CleanupJob{
			{
				JobID:       "running-job",
				JobType:     "automatic",
				SignalType:  "logs",
				StartedAt:   started,
				Status:      "running",
				CompletedAt: nil,
			},
			{
				JobID:          "completed-job",
				JobType:        "automatic",
				SignalType:     "traces",
				StartedAt:      started,
				CompletedAt:    &completed,
				Status:         "completed",
				RecordsDeleted: 7,
			},
		},
	}

	rec := serveRetentionListJobs(t, repo)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload cleanupJobsResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 2 || len(payload.Jobs) != 2 {
		t.Fatalf("payload = %#v, want two jobs and total 2", payload)
	}
	if payload.Jobs[0].CompletedAt != nil {
		t.Fatalf("running job completedAt = %#v, want nil", payload.Jobs[0].CompletedAt)
	}
	if payload.Jobs[1].CompletedAt == nil || *payload.Jobs[1].CompletedAt != "2026-05-11T12:02:00Z" {
		t.Fatalf("completed job completedAt = %#v", payload.Jobs[1].CompletedAt)
	}
}

func TestRetentionListJobs_ReturnsJSONError(t *testing.T) {
	rec := serveRetentionListJobs(t, &retentionHandlerRepo{listErr: errors.New("boom")})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Unable to load cleanup jobs") {
		t.Fatalf("body = %q, want cleanup jobs error", body)
	}
}
