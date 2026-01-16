package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"opentelemetry-dashboard/backend/internal/query"
)

// TraceSpansHandler returns spans for a trace.
type TraceSpansHandler struct {
	Service *query.TraceSpansService
}

func (h *TraceSpansHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceId")
	if traceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if h.Service == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	result, err := h.Service.Spans(r.Context(), traceID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
