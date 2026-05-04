package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/retention"
)

type RetentionHandler struct {
	Repo    retention.Repository
	Service *retention.CleanupService
}

type retentionSettingsResponse struct {
	Settings []retentionSettingItem `json:"settings"`
}

type retentionSettingItem struct {
	SignalType    string `json:"signalType"`
	RetentionDays uint32 `json:"retentionDays"`
}

type updateRetentionRequest struct {
	SignalType    string `json:"signalType"`
	RetentionDays uint32 `json:"retentionDays"`
}

const (
	minRetentionDays      uint32 = 1
	maxRetentionDays      uint32 = 365
	maxTraceRetentionDays uint32 = 15
)

type cleanupRequest struct {
	SignalTypes []string `json:"signalTypes"`
	ServiceName string   `json:"serviceName"`
}

type cleanupResponse struct {
	JobID   string                    `json:"jobId"`
	Results []retention.CleanupResult `json:"results"`
}

type cleanupJobsResponse struct {
	Jobs  []cleanupJobItem `json:"jobs"`
	Total uint64           `json:"total"`
}

type cleanupJobItem struct {
	JobID          string  `json:"jobId"`
	JobType        string  `json:"jobType"`
	SignalType     string  `json:"signalType"`
	ServiceName    string  `json:"serviceName"`
	StartedAt      string  `json:"startedAt"`
	CompletedAt    *string `json:"completedAt"`
	Status         string  `json:"status"`
	RecordsDeleted uint64  `json:"recordsDeleted"`
	ErrorMessage   string  `json:"errorMessage"`
}

func (h *RetentionHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	settings, err := h.Repo.GetSettings(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	items := make([]retentionSettingItem, len(settings))
	for i, s := range settings {
		items[i] = retentionSettingItem{
			SignalType:    s.SignalType,
			RetentionDays: s.RetentionDays,
		}
	}

	writeJSON(w, retentionSettingsResponse{Settings: items})
}

func (h *RetentionHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var req updateRetentionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.SignalType != "logs" && req.SignalType != "traces" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.RetentionDays < minRetentionDays || req.RetentionDays > maxRetentionDays {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.SignalType == "traces" && req.RetentionDays > maxTraceRetentionDays {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID := auth.UserID(r.Context())
	if err := h.Repo.UpdateSetting(r.Context(), req.SignalType, req.RetentionDays, userID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RetentionHandler) ManualCleanup(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var req cleanupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(req.SignalTypes) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, signalType := range req.SignalTypes {
		if signalType != "logs" && signalType != "traces" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	var results []retention.CleanupResult
	var err error

	if req.ServiceName == "" {
		results, err = h.Service.CleanupAll(r.Context(), req.SignalTypes)
	} else {
		results, err = h.Service.CleanupByService(r.Context(), req.SignalTypes, req.ServiceName)
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jobID := ""
	if len(results) > 0 {
		jobID = results[0].JobID
	}

	writeJSON(w, cleanupResponse{
		JobID:   jobID,
		Results: results,
	})
}

func (h *RetentionHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if r.URL.Query().Get("all") == "1" {
		limit = 0
	}

	jobs, err := h.Repo.ListCleanupJobs(r.Context(), limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	total, err := h.Repo.CountCleanupJobs(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	items := make([]cleanupJobItem, len(jobs))
	for i, j := range jobs {
		var completedAt *string
		if j.CompletedAt != nil {
			t := j.CompletedAt.Format("2006-01-02T15:04:05Z")
			completedAt = &t
		}

		items[i] = cleanupJobItem{
			JobID:          j.JobID,
			JobType:        j.JobType,
			SignalType:     j.SignalType,
			ServiceName:    j.ServiceName,
			StartedAt:      j.StartedAt.Format("2006-01-02T15:04:05Z"),
			CompletedAt:    completedAt,
			Status:         j.Status,
			RecordsDeleted: j.RecordsDeleted,
			ErrorMessage:   j.ErrorMessage,
		}
	}

	writeJSON(w, cleanupJobsResponse{Jobs: items, Total: total})
}
