package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/infrastructure/config"
)

type AttributesHandler struct {
	Service *query.Service
	Timeout time.Duration
}

func (h *AttributesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = time.Duration(config.Auto().QueryAttributesTimeoutSec) * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	search := r.URL.Query().Get("q")
	// Start with only logs context for now as requested
	keys, err := h.Service.GetLogAttributeKeys(ctx, search)
	if err != nil {
		log.Printf("query.attributes failed: %v", err)
		if ctx.Err() == context.DeadlineExceeded {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}
