package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"opentelemetry-dashboard/backend/internal/query"
)

// QueryHandler runs ad-hoc queries.
type QueryHandler struct {
	Service *query.Service
}

func (h *QueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req query.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("query.run decode failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf(
		"query.run request: signals=%v from=%s to=%s filters=%d page=%d limit=%d orderBy=%s",
		req.Signals,
		req.TimeRange.From.Format(time.RFC3339),
		req.TimeRange.To.Format(time.RFC3339),
		len(req.Filters),
		req.Page,
		req.Limit,
		req.OrderBy,
	)
	result, err := h.Service.Run(r.Context(), req)
	if err != nil {
		log.Printf("query.run failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	log.Printf("query.run success: runId=%s status=%s", result.RunID, result.Status)
	json.NewEncoder(w).Encode(result)
}
