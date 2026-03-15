package handlers

import (
	"encoding/json"
	"net/http"

	"opendashly/backend/internal/application/status"
)

type StatusHandler struct {
	Service *status.Service
}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	summary := h.Service.Summary(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *StatusHandler) ServeRuntimeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	runtimeSummary := h.Service.Runtime(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(runtimeSummary); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
