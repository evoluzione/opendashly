package ai

import "context"

type Service struct {
	Repo Repository
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
