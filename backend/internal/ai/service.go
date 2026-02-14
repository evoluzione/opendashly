package ai

import (
	"context"
	"strings"
)

type Service struct {
	Repo Repository
}

func (s *Service) GetSettings(ctx context.Context, tenantID string) (*Settings, error) {
	settings, err := s.Repo.GetSettings(ctx, tenantID)
	if err != nil {
		// If not found, return default settings
		return &Settings{
			TenantID: tenantID,
			Enabled:  false,
			Provider: "openai",
			Model:    "gpt-3.5-turbo",
			APIKey:   "",
		}, nil
	}
	return settings, nil
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID string, req UpdateSettingsRequest) (*Settings, error) {
	// Simple validation
	if req.Provider == "" {
		req.Provider = "openai"
	}
	if req.Model == "" {
		req.Model = "gpt-3.5-turbo"
	}
	req.APIKey = strings.TrimSpace(req.APIKey)

	// Preserve current API key when UI sends masked placeholder or empty value.
	if req.APIKey == "" || req.APIKey == "******" {
		existing, err := s.Repo.GetSettings(ctx, tenantID)
		if err == nil {
			req.APIKey = existing.APIKey
		}
	}

	settings := Settings{
		TenantID: tenantID,
		Enabled:  req.Enabled,
		Provider: req.Provider,
		Model:    req.Model,
		APIKey:   req.APIKey,
	}

	if err := s.Repo.UpsertSettings(ctx, settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func (s *Service) GetAssistantSession(ctx context.Context, tenantID, userID string) (*AssistantSession, error) {
	session, err := s.Repo.GetAssistantSession(ctx, tenantID, userID)
	if err != nil {
		// No saved session yet.
		return &AssistantSession{
			TenantID:     tenantID,
			UserID:       userID,
			MessagesJSON: "[]",
		}, nil
	}
	return session, nil
}

func (s *Service) SaveAssistantSession(ctx context.Context, tenantID, userID, messagesJSON string) error {
	return s.Repo.UpsertAssistantSession(ctx, AssistantSession{
		TenantID:     tenantID,
		UserID:       userID,
		MessagesJSON: messagesJSON,
	})
}

func (s *Service) ResetAssistantSession(ctx context.Context, tenantID, userID string) error {
	return s.Repo.DeleteAssistantSession(ctx, tenantID, userID)
}
