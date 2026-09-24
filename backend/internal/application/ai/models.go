package ai

import (
	"encoding/json"
	"time"
)

type AssistantMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// Context is the diagnosis context of an assistant answer, kept so that
	// follow-ups still work after the chat is reloaded.
	Context json.RawMessage `json:"context,omitempty"`
	// Suggestions are the follow-up buttons of an answer, shown again after a reload.
	Suggestions json.RawMessage `json:"suggestions,omitempty"`
	// Links open or download the data behind an answer.
	Links json.RawMessage `json:"links,omitempty"`
	// Rows are the actions of each numbered item of an answer.
	Rows json.RawMessage `json:"rows,omitempty"`
}

type AssistantSession struct {
	TenantID     string    `json:"tenantId"`
	UserID       string    `json:"userId"`
	MessagesJSON string    `json:"messagesJson"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
