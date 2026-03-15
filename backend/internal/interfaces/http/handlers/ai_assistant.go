package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"opendashly/backend/internal/application/ai"
	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/query"
)

type AIAvailabilityHandler struct {
	Service *ai.Service
}

func (h *AIAvailabilityHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Service.GetSettings(r.Context(), "default")
	if err != nil {
		http.Error(w, "unable to read ai settings", http.StatusInternalServerError)
		return
	}
	enabled := settings.Enabled && settings.APIKey != ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"enabled": enabled})
}

type AIAssistantChatHandler struct {
	AIService    *ai.Service
	QueryService *query.Service
}

type aiAssistantChatRequest struct {
	Prompt   string                   `json:"prompt"`
	Messages []query.AssistantMessage `json:"messages"`
}

func (h *AIAssistantChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req aiAssistantChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	settings, err := h.AIService.GetSettings(r.Context(), "default")
	if err != nil {
		http.Error(w, "unable to read ai settings", http.StatusInternalServerError)
		return
	}
	if !settings.Enabled || settings.APIKey == "" {
		http.Error(w, "Assistente AI non disponibile: abilita Smart Search e configura API Key", http.StatusForbidden)
		return
	}

	resp, err := query.RunAssistantChat(
		r.Context(),
		settings,
		h.QueryService,
		req.Prompt,
		req.Messages,
		time.Now().UTC(),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type aiAssistantSessionPayload struct {
	Messages []query.AssistantMessage `json:"messages"`
}

type AIAssistantSessionHandler struct {
	AIService *ai.Service
}

func (h *AIAssistantSessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	session, err := h.AIService.GetAssistantSession(r.Context(), "default", userID)
	if err != nil {
		http.Error(w, "unable to read assistant session", http.StatusInternalServerError)
		return
	}

	var payload aiAssistantSessionPayload
	if err := json.Unmarshal([]byte(session.MessagesJSON), &payload.Messages); err != nil {
		payload.Messages = []query.AssistantMessage{}
	}
	if payload.Messages == nil {
		payload.Messages = []query.AssistantMessage{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

func (h *AIAssistantSessionHandler) Put(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var payload aiAssistantSessionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if payload.Messages == nil {
		payload.Messages = []query.AssistantMessage{}
	}

	raw, err := json.Marshal(payload.Messages)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := h.AIService.SaveAssistantSession(r.Context(), "default", userID, string(raw)); err != nil {
		http.Error(w, "unable to save assistant session", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AIAssistantSessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := h.AIService.ResetAssistantSession(r.Context(), "default", userID); err != nil {
		http.Error(w, "unable to reset assistant session", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
