package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"opendashly/backend/internal/application/query"
)

type AttributesHandler struct {
	Service *query.Service
}

func (h *AttributesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	// Start with only logs context for now as requested
	keys, err := h.Service.GetLogAttributeKeys(r.Context(), search)
	if err != nil {
		log.Printf("query.attributes failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}
