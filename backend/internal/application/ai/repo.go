package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Repository interface {
	GetSettings(ctx context.Context, tenantID string) (*Settings, error)
	UpsertSettings(ctx context.Context, settings Settings) error
	GetAssistantSession(ctx context.Context, tenantID, userID string) (*AssistantSession, error)
	UpsertAssistantSession(ctx context.Context, session AssistantSession) error
	DeleteAssistantSession(ctx context.Context, tenantID, userID string) error
}

type Repo struct {
	Conn driver.Conn
}

func (r *Repo) GetSettings(ctx context.Context, tenantID string) (*Settings, error) {
	query := `SELECT tenant_id, enabled, provider, model, api_key, updated_at
	          FROM telemetry.ai_settings
	          WHERE tenant_id = ?
	          ORDER BY updated_at DESC
	          LIMIT 1`

	var s Settings
	// enabled is stored as UInt8 (0 or 1), scan into bool might need custom handling or direct scan if driver supports it.
	// Clickhouse driver usually handles UInt8 -> bool. If not, we scan to uint8 and convert.
	// Let's safe bet: scan to bool directly, if it fails we fix.

	// Actually, for ClickHouse driver, UInt8 maps to uint8. better safe:
	var enabled uint8

	row := r.Conn.QueryRow(ctx, query, tenantID)
	if err := row.Scan(&s.TenantID, &enabled, &s.Provider, &s.Model, &s.APIKey, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.Enabled = enabled == 1

	return &s, nil
}

func (r *Repo) UpsertSettings(ctx context.Context, s Settings) error {
	query := `INSERT INTO telemetry.ai_settings (tenant_id, enabled, provider, model, api_key, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	var enabled uint8
	if s.Enabled {
		enabled = 1
	}

	err := r.Conn.Exec(ctx, query, s.TenantID, enabled, s.Provider, s.Model, s.APIKey, time.Now())
	if err != nil {
		return fmt.Errorf("upsert ai settings: %w", err)
	}
	return nil
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
