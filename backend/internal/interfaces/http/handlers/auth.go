package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"opendashly/backend/internal/application/auth"
)

type AuthHandler struct {
	Repo       auth.Repository
	Secret     []byte
	CookieName string
	SessionTTL time.Duration
	TenantID   string
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type firstLoginChangePasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

type sessionResponse struct {
	User               userResponse `json:"user"`
	MustChangePassword bool         `json:"mustChangePassword"`
}

type userResponse struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	IsDisabled bool   `json:"isDisabled"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	user, err := h.Repo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Credenziali non valide")
		return
	}
	if user.IsDisabled {
		writeError(w, http.StatusForbidden, "Utente disabilitato")
		return
	}
	if err := auth.ComparePassword(user.PasswordHash, req.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "Credenziali non valide")
		return
	}
	if err := h.Repo.UpdateLastLogin(r.Context(), user.ID, time.Now().UTC()); err != nil {
		log.Printf("auth.login: update last login failed user=%s err=%v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "Errore interno")
		return
	}
	token, err := auth.GenerateToken(h.Secret, user.ID, user.Role, h.SessionTTL)
	if err != nil {
		log.Printf("auth.login: generate token failed user=%s err=%v", user.ID, err)
		writeError(w, http.StatusInternalServerError, "Errore creazione sessione")
		return
	}
	setSessionCookie(w, h.CookieName, token, h.SessionTTL, r.TLS != nil)
	writeJSON(w, sessionResponse{User: toUserResponse(*user), MustChangePassword: user.MustChangePassword})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w, h.CookieName, r.TLS != nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := h.Repo.GetByID(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := auth.ComparePassword(user.PasswordHash, req.CurrentPassword); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("auth.change_password: hash failed user=%s err=%v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.Repo.UpdatePassword(r.Context(), user.ID, newHash, false); err != nil {
		log.Printf("auth.change_password: update password failed user=%s err=%v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	updated, err := h.Repo.GetByID(r.Context(), user.ID)
	if err != nil {
		log.Printf("auth.change_password: reload user failed user=%s err=%v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	token, err := auth.GenerateToken(h.Secret, updated.ID, updated.Role, h.SessionTTL)
	if err != nil {
		log.Printf("auth.change_password: generate token failed user=%s err=%v", updated.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, h.CookieName, token, h.SessionTTL, r.TLS != nil)
	writeJSON(w, sessionResponse{User: toUserResponse(*updated), MustChangePassword: updated.MustChangePassword})
}

func (h *AuthHandler) FirstLoginChangePassword(w http.ResponseWriter, r *http.Request) {
	var req firstLoginChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := h.Repo.GetByID(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !user.MustChangePassword {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err := auth.ComparePassword(user.PasswordHash, "admin"); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("auth.first_login_change_password: hash failed user=%s err=%v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.Repo.UpdatePassword(r.Context(), user.ID, newHash, false); err != nil {
		log.Printf("auth.first_login_change_password: update password failed user=%s err=%v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	updated, err := h.Repo.GetByID(r.Context(), user.ID)
	if err != nil {
		log.Printf("auth.first_login_change_password: reload user failed user=%s err=%v", user.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	token, err := auth.GenerateToken(h.Secret, updated.ID, updated.Role, h.SessionTTL)
	if err != nil {
		log.Printf("auth.first_login_change_password: generate token failed user=%s err=%v", updated.ID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	setSessionCookie(w, h.CookieName, token, h.SessionTTL, r.TLS != nil)
	writeJSON(w, sessionResponse{User: toUserResponse(*updated), MustChangePassword: updated.MustChangePassword})
}

func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	user, err := h.Repo.GetByID(r.Context(), userID)
	if err != nil {
		log.Printf("auth.session: user lookup failed user=%s err=%v", userID, err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	writeJSON(w, sessionResponse{User: toUserResponse(*user), MustChangePassword: user.MustChangePassword})
}

func toUserResponse(user auth.User) userResponse {
	return userResponse{
		ID:         user.ID,
		Username:   user.Username,
		Role:       user.Role,
		IsDisabled: user.IsDisabled,
	}
}

func setSessionCookie(w http.ResponseWriter, name, value string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("auth.write_json: encode failed err=%v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
