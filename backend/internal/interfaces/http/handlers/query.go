package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/application/pressure"
	"opendashly/backend/internal/application/query"
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
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	result, err := h.Service.Run(ctx, req)
	if err != nil {
		log.Printf("query.run failed: %v", err)
		if errors.Is(err, pressure.ErrBusy) {
			writeError(w, http.StatusServiceUnavailable, "Backend sotto pressione: restringi intervallo o filtri e riprova.")
			return
		}
		if ctx.Err() == context.DeadlineExceeded {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	log.Printf("query.run success: runId=%s status=%s", result.RunID, result.Status)
	json.NewEncoder(w).Encode(result)
}
