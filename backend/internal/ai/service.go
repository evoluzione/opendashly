package ai

import (
	"context"
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
