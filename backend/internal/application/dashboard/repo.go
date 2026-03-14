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

func (r *Repo) ensureVersionColumn(ctx context.Context) error {
	const q = `ALTER TABLE telemetry.dashboard_settings
		ADD COLUMN IF NOT EXISTS version UInt64 DEFAULT 0`
	if err := r.Conn.Exec(ctx, q); err != nil {
		return fmt.Errorf("ensure dashboard_settings.version column: %w", err)
	}
	return nil
}

func (r *Repo) GetSettings(ctx context.Context, tenantID string) ([]ChartSetting, error) {
	if err := r.ensureVersionColumn(ctx); err != nil {
		return nil, err
	}

	const query = `SELECT chart_key,
		argMax(enabled, version) AS enabled,
		argMax(order_index, version) AS order_index,
		argMax(x, version) AS x,
		argMax(y, version) AS y,
		argMax(w, version) AS w,
		argMax(h, version) AS h
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
		var x int32
		var y int32
		var w int32
		var h int32
		if err := rows.Scan(&key, &enabled, &orderIndex, &x, &y, &w, &h); err != nil {
			return nil, fmt.Errorf("scan dashboard settings: %w", err)
		}
		settings = append(settings, ChartSetting{
			Key:     key,
			Enabled: enabled == 1,
			Order:   orderIndex,
			X:       x,
			Y:       y,
			W:       w,
			H:       h,
		})
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
	if err := r.ensureVersionColumn(ctx); err != nil {
		return err
	}

	const query = `INSERT INTO telemetry.dashboard_settings (tenant_id, chart_key, enabled, order_index, x, y, w, h, version, updated_at, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	baseVersion := uint64(now.UnixNano())

	for i, setting := range settings {
		var enabled uint8
		if setting.Enabled {
			enabled = 1
		}
		version := baseVersion + uint64(i)

		err := r.Conn.Exec(
			ctx,
			query,
			tenantID,
			setting.Key,
			enabled,
			setting.Order,
			setting.X,
			setting.Y,
			setting.W,
			setting.H,
			version,
			now,
			updatedBy,
		)
		if err != nil {
			return fmt.Errorf("upsert dashboard settings: %w", err)
		}
	}
	return nil
}
