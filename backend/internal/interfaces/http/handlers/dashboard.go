package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/application/metrics"
)

// DashboardHandler handles dashboard metrics requests.
type DashboardHandler struct {
	Service *metrics.Service
	Timeout time.Duration
}

type dashboardRequest struct {
	From        string `json:"from"`
	To          string `json:"to"`
	ServiceName string `json:"serviceName,omitempty"`
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req dashboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("dashboard.metrics decode failed: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default time range: last 6 hours
	defaultTo := time.Now().UTC()
	defaultFrom := defaultTo.Add(-6 * time.Hour)

	var from, to time.Time
	var err error

	if req.From == "" {
		from = defaultFrom
	} else {
		from, err = time.Parse(time.RFC3339, req.From)
		if err != nil {
			log.Printf("dashboard.metrics invalid from time: %v, using default", err)
			from = defaultFrom
		}
	}

	if req.To == "" {
		to = defaultTo
	} else {
		to, err = time.Parse(time.RFC3339, req.To)
		if err != nil {
			log.Printf("dashboard.metrics invalid to time: %v, using default", err)
			to = defaultTo
		}
	}

	if !to.After(from) {
		log.Printf("dashboard.metrics invalid range: from=%s to=%s, using default last 6h",
			from.Format(time.RFC3339), to.Format(time.RFC3339))
		from = defaultFrom
		to = defaultTo
	}

	log.Printf("dashboard.metrics request: from=%s to=%s service=%s",
		from.Format(time.RFC3339), to.Format(time.RFC3339), req.ServiceName)

	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	result, err := h.Service.GetDashboard(ctx, metrics.DashboardRequest{
		From:        from,
		To:          to,
		ServiceName: req.ServiceName,
	})
	if err != nil {
		log.Printf("dashboard.metrics failed: %v", err)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			http.Error(w, "Gateway timeout", http.StatusGatewayTimeout)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	log.Printf("dashboard.metrics success")
	json.NewEncoder(w).Encode(result)
}
