package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/application/query"
)

type ServicesHandler struct {
	Service *query.Service
	Timeout time.Duration
}

type servicesResponse struct {
	Services []string `json:"services"`
}

func (h *ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	services, err := h.Service.ListServices(ctx)
	if err != nil {
		log.Printf("services.list failed: %v", err)
		if ctx.Err() == context.DeadlineExceeded {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, servicesResponse{Services: services})
}
