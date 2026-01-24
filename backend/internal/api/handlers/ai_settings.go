package handlers

import (
	"encoding/json"
	"net/http"

	"opendashly/backend/internal/ai"
)

type AISettingsHandler struct {
	Service *ai.Service
}

func (h *AISettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	// Assuming tenantID is handled via middleware or hardcoded to "default" for now
	tenantID := "default"

	settings, err := h.Service.GetSettings(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mask API Key for security
	masked := *settings
	if masked.APIKey != "" {
		masked.APIKey = "******"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(masked)
}

func (h *AISettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := "default"

	var req ai.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	updated, err := h.Service.UpdateSettings(r.Context(), tenantID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Mask ID in response
	masked := *updated
	if masked.APIKey != "" {
		masked.APIKey = "******"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(masked)
}
