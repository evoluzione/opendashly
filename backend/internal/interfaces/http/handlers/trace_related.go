package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"opendashly/backend/internal/application/query"
)

// TraceRelatedHandler returns related telemetry for a trace.
type TraceRelatedHandler struct {
	Service *query.RelatedService
}

func (h *TraceRelatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	traceID := chi.URLParam(r, "traceId")
	if traceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := h.Service.Related(r.Context(), traceID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
