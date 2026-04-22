package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const keyTitle = "title"

type Repository interface {
	GetSetting(ctx context.Context, tenantID, key string) (string, error)
	UpsertSetting(ctx context.Context, tenantID, key, value, updatedBy string) error
}

type Repo struct {
	Conn driver.Conn
}

func (r *Repo) GetSetting(ctx context.Context, tenantID, key string) (string, error) {
	const query = `SELECT setting_value
		FROM telemetry.workspace_settings
		WHERE tenant_id = ? AND setting_key = ?
		ORDER BY updated_at DESC
		LIMIT 1`

	var value string
	row := r.Conn.QueryRow(ctx, query, tenantID, key)
	if err := row.Scan(&value); err != nil {
		return "", err
	}
	return value, nil
}

func (r *Repo) UpsertSetting(ctx context.Context, tenantID, key, value, updatedBy string) error {
	const query = `INSERT INTO telemetry.workspace_settings (tenant_id, setting_key, setting_value, updated_at, updated_by)
		VALUES (?, ?, ?, ?, ?)`
	if err := r.Conn.Exec(ctx, query, tenantID, key, value, time.Now(), updatedBy); err != nil {
		return fmt.Errorf("upsert workspace setting: %w", err)
	}
	return nil
}

type Service struct {
	Repo Repository
}

func (s *Service) GetSettings(ctx context.Context, tenantID string) (*Settings, error) {
	title, err := s.Repo.GetSetting(ctx, tenantID, keyTitle)
	if err != nil {
		if isNoRowsError(err) {
			return &Settings{Title: DefaultTitle}, nil
		}
		return nil, err
	}
	if title == "" {
		title = DefaultTitle
	}
	return &Settings{Title: title}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID, updatedBy string, req UpdateSettingsRequest) (*Settings, error) {
	title := req.Title
	if title == "" {
		title = DefaultTitle
	}
	if err := s.Repo.UpsertSetting(ctx, tenantID, keyTitle, title, updatedBy); err != nil {
		return nil, err
	}
	return &Settings{Title: title}, nil
}

func isNoRowsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no rows") || strings.Contains(msg, "eof")
}
