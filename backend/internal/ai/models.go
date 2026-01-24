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
