package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"opendashly/backend/internal/query"
)

type SmartQueryHandler struct{}

type smartQueryRequest struct {
	Prompt string `json:"prompt"`
}

func (h *SmartQueryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req smartQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	response, err := query.BuildSmartQuery(req.Prompt, time.Now())
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
