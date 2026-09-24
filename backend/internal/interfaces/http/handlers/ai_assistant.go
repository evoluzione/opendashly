package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
	// The alpine image has no zoneinfo: embed it for the viewer's time zone.
	_ "time/tzdata"

	"opendashly/backend/internal/application/ai"
	"opendashly/backend/internal/application/auth"
	"opendashly/backend/internal/application/diagnosis"
)

// AIAssistantChatHandler runs the deterministic diagnosis on the prompt.
type AIAssistantChatHandler struct {
	Runner *diagnosis.Runner
}

type aiAssistantChatRequest struct {
	Prompt string `json:"prompt"`
	Locale string `json:"locale"`
	// Context is the previous answer's context, echoed back by the UI.
	Context *diagnosis.Context `json:"context"`
	// TimeZone is the viewer's IANA zone ("Europe/Rome"): "oggi" and "ieri"
	// start at the viewer's midnight, and times are shown in that zone.
	TimeZone string `json:"timeZone"`
}

// diagnosisTimeout stays below the server WriteTimeout (at least 30s).
const diagnosisTimeout = 25 * time.Second

func (h *AIAssistantChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req aiAssistantChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), diagnosisTimeout)
	defer cancel()
	now := time.Now().UTC()
	if loc, err := time.LoadLocation(req.TimeZone); err == nil && req.TimeZone != "" {
		now = now.In(loc)
	}
	resp, err := h.Runner.Run(ctx, req.Prompt, req.Locale, now, req.Context)
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
	Messages []ai.AssistantMessage `json:"messages"`
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
		payload.Messages = []ai.AssistantMessage{}
	}
	if payload.Messages == nil {
		payload.Messages = []ai.AssistantMessage{}
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
		payload.Messages = []ai.AssistantMessage{}
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
