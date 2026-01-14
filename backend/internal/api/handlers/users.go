package handlers

import (
	"encoding/json"
	"net/http"

	"opentelemetry-dashboard/backend/internal/auth"
)

type UsersHandler struct {
	Repo auth.Repository
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	users, err := h.Repo.List(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	responses := make([]userResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, toUserResponse(user))
	}
	writeJSON(w, responses)
}

func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if req.Role != auth.RoleAdmin && req.Role != auth.RoleUser {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := h.Repo.GetByUsername(r.Context(), req.Username)
	if err == nil {
		w.WriteHeader(http.StatusConflict)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	created, err := h.Repo.Create(r.Context(), auth.User{
		Username:           req.Username,
		PasswordHash:       hash,
		Role:               req.Role,
		MustChangePassword: false,
		IsDisabled:         false,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, toUserResponse(*created))
}

func isAdmin(r *http.Request) bool {
	return auth.Role(r.Context()) == auth.RoleAdmin
}
