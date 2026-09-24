package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Repository interface {
	GetAssistantSession(ctx context.Context, tenantID, userID string) (*AssistantSession, error)
	UpsertAssistantSession(ctx context.Context, session AssistantSession) error
	DeleteAssistantSession(ctx context.Context, tenantID, userID string) error
}

type Repo struct {
	Conn driver.Conn
}

func (r *Repo) GetAssistantSession(ctx context.Context, tenantID, userID string) (*AssistantSession, error) {
	query := `SELECT tenant_id, user_id, messages_json, updated_at
	          FROM telemetry.ai_assistant_sessions
	          WHERE tenant_id = ? AND user_id = ?
	          ORDER BY updated_at DESC
	          LIMIT 1`

	var session AssistantSession
	row := r.Conn.QueryRow(ctx, query, tenantID, userID)
	if err := row.Scan(&session.TenantID, &session.UserID, &session.MessagesJSON, &session.UpdatedAt); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *Repo) UpsertAssistantSession(ctx context.Context, session AssistantSession) error {
	query := `INSERT INTO telemetry.ai_assistant_sessions (tenant_id, user_id, messages_json, updated_at)
	          VALUES (?, ?, ?, ?)`
	if err := r.Conn.Exec(ctx, query, session.TenantID, session.UserID, session.MessagesJSON, time.Now().UTC()); err != nil {
		return fmt.Errorf("upsert ai assistant session: %w", err)
	}
	return nil
}

func (r *Repo) DeleteAssistantSession(ctx context.Context, tenantID, userID string) error {
	query := `ALTER TABLE telemetry.ai_assistant_sessions DELETE WHERE tenant_id = ? AND user_id = ?`
	if err := r.Conn.Exec(ctx, query, tenantID, userID); err != nil {
		return fmt.Errorf("delete ai assistant session: %w", err)
	}
	return nil
}
