package handlers

import (
	"encoding/json"
	"net/http"

	"opentelemetry-dashboard/backend/internal/query"
)

// QueryHandler runs ad-hoc queries.
type QueryHandler struct {
	Service *query.Service
}

func (h *QueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req query.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := h.Service.Run(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
