package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Repository interface {
	GetSettings(ctx context.Context, tenantID string) ([]ChartSetting, error)
	UpsertSettings(ctx context.Context, tenantID string, settings []ChartSetting, updatedBy string) error
}

type Repo struct {
	Conn driver.Conn
}

func (r *Repo) GetSettings(ctx context.Context, tenantID string) ([]ChartSetting, error) {
	query := `SELECT chart_key,
		argMax(enabled, updated_at) AS enabled,
		argMax(order_index, updated_at) AS order_index
		FROM telemetry.dashboard_settings
		WHERE tenant_id = ?
		GROUP BY chart_key`

	rows, err := r.Conn.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get dashboard settings: %w", err)
	}
	defer rows.Close()

	var settings []ChartSetting
	for rows.Next() {
		var key string
		var enabled uint8
		var orderIndex int32
		if err := rows.Scan(&key, &enabled, &orderIndex); err != nil {
			return nil, fmt.Errorf("scan dashboard settings: %w", err)
		}
		settings = append(settings, ChartSetting{Key: key, Enabled: enabled == 1, Order: orderIndex})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dashboard settings: %w", err)
	}
	if settings == nil {
		settings = []ChartSetting{}
	}
	return settings, nil
}

func (r *Repo) UpsertSettings(ctx context.Context, tenantID string, settings []ChartSetting, updatedBy string) error {
	query := `INSERT INTO telemetry.dashboard_settings (tenant_id, chart_key, enabled, order_index, updated_at, updated_by)
		VALUES (?, ?, ?, ?, ?, ?)`

	now := time.Now()
	for _, setting := range settings {
		var enabled uint8
		if setting.Enabled {
			enabled = 1
		}
		if err := r.Conn.Exec(ctx, query, tenantID, setting.Key, enabled, setting.Order, now, updatedBy); err != nil {
			return fmt.Errorf("upsert dashboard settings: %w", err)
		}
	}
	return nil
}
