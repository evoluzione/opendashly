package ai

import "time"

type Settings struct {
	TenantID  string    `json:"tenantId"`
	Enabled   bool      `json:"enabled"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	APIKey    string    `json:"apiKey"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UpdateSettingsRequest struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

type AssistantSession struct {
	TenantID     string    `json:"tenantId"`
	UserID       string    `json:"userId"`
	MessagesJSON string    `json:"messagesJson"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
