package handlers

import (
	"encoding/json"
	"net/http"

	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/dashboard"
)

type DashboardSettingsHandler struct {
	Service *dashboard.Service
}

func (h *DashboardSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantID(r.Context())
	if tenantID == "" {
		tenantID = "default"
	}

	settings, err := h.Service.GetSettings(r.Context(), tenantID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, dashboard.SettingsResponse{Settings: settings})
}

func (h *DashboardSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var req dashboard.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tenantID := auth.TenantID(r.Context())
	if tenantID == "" {
		tenantID = "default"
	}
	userID := auth.UserID(r.Context())
	settings, err := h.Service.UpdateSettings(r.Context(), tenantID, userID, req.Settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, dashboard.SettingsResponse{Settings: settings})
}
