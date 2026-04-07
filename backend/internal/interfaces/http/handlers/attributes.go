package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/application/query"
)

type AttributesHandler struct {
	Service *query.Service
}

func (h *AttributesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
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
