package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/retention"
)

type RetentionHandler struct {
	Repo                  retention.Repository
	Service               *retention.CleanupService
	MaxLogRetentionDays   uint32
	MaxTraceRetentionDays uint32
}

type retentionSettingsResponse struct {
	Settings []retentionSettingItem `json:"settings"`
}

type retentionSettingItem struct {
	SignalType       string `json:"signalType"`
	RetentionDays    uint32 `json:"retentionDays"`
	MaxRetentionDays uint32 `json:"maxRetentionDays"`
}

type updateRetentionRequest struct {
	SignalType    string `json:"signalType"`
	RetentionDays uint32 `json:"retentionDays"`
}

const (
	minRetentionDays             uint32 = 1
	defaultMaxLogRetentionDays   uint32 = 30
	defaultMaxTraceRetentionDays uint32 = 15
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

type emergencyStateResponse struct {
	Enabled              bool      `json:"enabled"`
	Mode                 string    `json:"mode"`
	Level                int       `json:"level"`
	LastTransitionAt     string    `json:"lastTransitionAt"`
	LastPressureAt       *string   `json:"lastPressureAt,omitempty"`
	PressureCooldownEnds *string   `json:"pressureCooldownEnds,omitempty"`
	LastReasons          []string  `json:"lastReasons"`
	Snapshot             snapshot  `json:"snapshot"`
}

type snapshot struct {
	At               string   `json:"at"`
	MemoryUsageMiB   uint64   `json:"memoryUsageMiB"`
	MemoryThreshold  uint64   `json:"memoryThresholdMiB"`
	MemoryPercent    float64  `json:"memoryPercent"`
	DiskUsedBytes    uint64   `json:"diskUsedBytes"`
	DiskTotalBytes   uint64   `json:"diskTotalBytes"`
	DiskUsagePercent float64  `json:"diskUsagePercent"`
	RecentErrors     int      `json:"recentErrors"`
	Reasons          []string `json:"reasons"`
	Pressure         bool     `json:"pressure"`
}

type updateEmergencyRequest struct {
	Enabled bool `json:"enabled"`
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
			SignalType:       s.SignalType,
			RetentionDays:    s.RetentionDays,
			MaxRetentionDays: h.maxRetentionForSignal(s.SignalType),
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
	maxForSignal := h.maxRetentionForSignal(req.SignalType)
	if req.RetentionDays < minRetentionDays || req.RetentionDays > maxForSignal {
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

func (h *RetentionHandler) maxRetentionForSignal(signalType string) uint32 {
	if signalType == "traces" {
		if h.MaxTraceRetentionDays > 0 {
			return h.MaxTraceRetentionDays
		}
		return defaultMaxTraceRetentionDays
	}
	if h.MaxLogRetentionDays > 0 {
		return h.MaxLogRetentionDays
	}
	return defaultMaxLogRetentionDays
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
		log.Printf("retention.jobs: list failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Unable to load cleanup jobs")
		return
	}
	total, err := h.Repo.CountCleanupJobs(r.Context())
	if err != nil {
		log.Printf("retention.jobs: count failed: %v", err)
		writeError(w, http.StatusInternalServerError, "Unable to load cleanup jobs")
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

func (h *RetentionHandler) GetEmergencyState(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	state := h.Service.EmergencyState(r.Context())
	var lastPressure *string
	if state.LastPressureAt != nil {
		v := state.LastPressureAt.Format(time.RFC3339)
		lastPressure = &v
	}
	var cooldownEnd *string
	if state.PressureCooldownEnds != nil {
		v := state.PressureCooldownEnds.Format(time.RFC3339)
		cooldownEnd = &v
	}

	writeJSON(w, emergencyStateResponse{
		Enabled:              state.Enabled,
		Mode:                 state.Mode,
		Level:                state.Level,
		LastTransitionAt:     state.LastTransitionAt.Format(time.RFC3339),
		LastPressureAt:       lastPressure,
		PressureCooldownEnds: cooldownEnd,
		LastReasons:          state.LastReasons,
		Snapshot: snapshot{
			At:               state.Snapshot.At.Format(time.RFC3339),
			MemoryUsageMiB:   state.Snapshot.MemoryUsageMiB,
			MemoryThreshold:  state.Snapshot.MemoryThreshold,
			MemoryPercent:    state.Snapshot.MemoryPercent,
			DiskUsedBytes:    state.Snapshot.DiskUsedBytes,
			DiskTotalBytes:   state.Snapshot.DiskTotalBytes,
			DiskUsagePercent: state.Snapshot.DiskUsagePercent,
			RecentErrors:     state.Snapshot.RecentErrors,
			Reasons:          state.Snapshot.Reasons,
			Pressure:         state.Snapshot.Pressure,
		},
	})
}

func (h *RetentionHandler) UpdateEmergencyState(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var req updateEmergencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.Service.SetEmergencyEnabled(req.Enabled)
	h.GetEmergencyState(w, r)
}
