package handlers

import (
	"context"
	"net/http"
	"time"

	"opendashly/backend/internal/application/query"
)

type ServicesHandler struct {
	Service *query.Service
}

type servicesResponse struct {
	Services []string `json:"services"`
}

func (h *ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	services, err := h.Service.ListServices(ctx)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, servicesResponse{Services: services})
}
