package handlers

import (
	"net/http"

	"opendashly/backend/internal/application/query"
)

type ServicesHandler struct {
	Service *query.Service
}

type servicesResponse struct {
	Services []string `json:"services"`
}

func (h *ServicesHandler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.Service.ListServices(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, servicesResponse{Services: services})
}
