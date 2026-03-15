package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"opendashly/backend/internal/application/ai"
	"opendashly/backend/internal/application/query"
)

type SmartQueryHandler struct {
	AIService *ai.Service
}

type smartQueryRequest struct {
	Prompt      string `json:"prompt"`
	ContextType string `json:"contextType"` // "logs" or "metrics"
}

func (h *SmartQueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req smartQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Fetch dynamic settings
	// tenantID hardcoded for now
	settings, err := h.AIService.GetSettings(r.Context(), "default")
	if err != nil {
		// Log error but maybe proceed?
		// If DB fails, we might fall back to config?
		// For now fail safe -> proceed with empty settings (disabled)
		settings = &ai.Settings{Enabled: false}
	}

	// If disabled, return error or empty?
	// User Requirement: "l'amministratore deve poter attivare la ricerca smart che di default è disattiva"
	if !settings.Enabled || settings.APIKey == "" {
		http.Error(w, "Smart search is disabled or missing configuration", http.StatusForbidden)
		return
	}

	response, err := query.BuildSmartQuery(req.Prompt, req.ContextType, settings, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
