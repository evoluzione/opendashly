package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"opendashly/backend/internal/application/query"
	"opendashly/backend/internal/infrastructure/config"
)

type SavedQueriesHandler struct {
	Repo   *query.SavedQueryRepo
	Runner *query.Service
}

type savedQueryCreateRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Request     query.QueryRequest `json:"request"`
}

func (h *SavedQueriesHandler) List(w http.ResponseWriter, r *http.Request) {
	items, _ := h.Repo.List(r.Context())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": items})
}

func (h *SavedQueriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req savedQueryCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	item, _ := h.Repo.Create(r.Context(), req.Name, req.Description, req.Request)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *SavedQueriesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "queryId")
	item, ok := h.Repo.Get(r.Context(), id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *SavedQueriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "queryId")
	h.Repo.Delete(r.Context(), id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SavedQueriesHandler) Run(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "queryId")
	item, ok := h.Repo.Get(r.Context(), id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(config.Auto().QueryRequestTimeoutSec)*time.Second)
	defer cancel()

	result, err := h.Runner.Run(ctx, item.Request)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
