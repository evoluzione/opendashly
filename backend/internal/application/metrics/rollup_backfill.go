package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"opendashly/backend/internal/infrastructure/config"
	"opendashly/backend/internal/infrastructure/storage"
)

const dashboardRollupBackfillName = "dashboard_v1"

// StartRollupBackfill warms dashboard rollups for recent historical data.
// Materialized views cover new writes; this only fills data that existed
// before the rollup migration was applied.
func (s *Service) StartRollupBackfill(ctx context.Context, hours int) {
	store := s.backfillStore()
	if store == nil || store.Conn == nil {
		return
	}
	if hours <= 0 {
		hours = config.Auto().DashboardRollupBackfillHours
	}

	go func() {
		if err := s.runRollupBackfill(ctx, hours); err != nil {
			log.Printf("metrics.rollup_backfill: stopped: %v", err)
		}
	}()
}

func (s *Service) runRollupBackfill(ctx context.Context, hours int) error {
	upper, err := s.rollupMigrationAppliedAt(ctx)
	if err != nil {
		return err
	}
	if upper.IsZero() {
		upper = time.Now().UTC()
	}
	upper = upper.UTC().Truncate(time.Minute)

	lower := upper.Add(-time.Duration(hours) * time.Hour).Truncate(time.Minute)
	if !lower.Before(upper) {
		return nil
	}

	chunkSize := time.Duration(config.Auto().DashboardRollupChunkMinutes) * time.Minute
	for start := lower; start.Before(upper); start = start.Add(chunkSize) {
		if reason, ok := s.dashboardPressureActive(); ok {
			return fmt.Errorf("backend pressure active before backfill chunk: %s", reason)
		}
		end := start.Add(chunkSize)
		if end.After(upper) {
			end = upper
		}
		done, err := s.backfillChunkCompleted(ctx, start, end)
		if err != nil {
			return err
		}
		if done {
			continue
		}
		if err := s.backfillChunkAdaptive(ctx, start, end, time.Minute); err != nil {
			return err
		}
		if err := s.markBackfillChunkCompleted(ctx, start, end); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) backfillChunkAdaptive(ctx context.Context, start, end time.Time, minChunk time.Duration) error {
	err := s.backfillChunk(ctx, start, end)
	if err == nil {
		return nil
	}
	if !isDashboardPressureError(err) {
		return err
	}
	if end.Sub(start) <= minChunk {
		s.openDashboardPressure(err)
		return err
	}

	mid := start.Add(end.Sub(start) / 2).Truncate(time.Minute)
	if !mid.After(start) || !mid.Before(end) {
		s.openDashboardPressure(err)
		return err
	}
	log.Printf("metrics.rollup_backfill: splitting pressured chunk start=%s end=%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err := s.backfillChunkAdaptive(ctx, start, mid, minChunk); err != nil {
		return err
	}
	if err := s.backfillChunkAdaptive(ctx, mid, end, minChunk); err != nil {
		return err
	}
	return nil
}

func (s *Service) rollupMigrationAppliedAt(ctx context.Context) (time.Time, error) {
	rows, err := s.backfillStore().Conn.Query(ctx, `
		SELECT max(applied_at)
		FROM telemetry.schema_migrations
		WHERE name = '014_dashboard_rollups.sql'
	`)
	if err != nil {
		return time.Time{}, fmt.Errorf("read rollup migration timestamp: %w", err)
	}
	defer rows.Close()

	var appliedAt time.Time
	if rows.Next() {
		if err := rows.Scan(&appliedAt); err != nil {
			return time.Time{}, fmt.Errorf("scan rollup migration timestamp: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return time.Time{}, fmt.Errorf("iterate rollup migration timestamp: %w", err)
	}
	return appliedAt, nil
}

func (s *Service) backfillChunkCompleted(ctx context.Context, start, end time.Time) (bool, error) {
	rows, err := s.backfillStore().Conn.Query(ctx, `
		SELECT count()
		FROM telemetry.dashboard_rollup_backfill_chunks
		WHERE rollup = ? AND chunk_start = ? AND chunk_end = ?
	`, dashboardRollupBackfillName, start, end)
	if err != nil {
		return false, fmt.Errorf("read backfill chunk marker: %w", err)
	}
	defer rows.Close()

	var count uint64
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return false, fmt.Errorf("scan backfill chunk marker: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate backfill chunk marker: %w", err)
	}
	return count > 0, nil
}

func (s *Service) markBackfillChunkCompleted(ctx context.Context, start, end time.Time) error {
	if err := s.backfillStore().Conn.Exec(ctx, `
		INSERT INTO telemetry.dashboard_rollup_backfill_chunks
			(rollup, chunk_start, chunk_end, completed_at)
		VALUES (?, ?, ?, ?)
	`, dashboardRollupBackfillName, start, end, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark backfill chunk: %w", err)
	}
	return nil
}

func (s *Service) backfillChunk(ctx context.Context, start, end time.Time) error {
	log.Printf("metrics.rollup_backfill: chunk start=%s end=%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	for _, table := range []string{traceServiceRollupTable, traceEndpointRollupTable, logLevelRollupTable} {
		stmt := fmt.Sprintf(
			"ALTER TABLE %s DELETE WHERE time_bucket >= '%s' AND time_bucket < '%s' SETTINGS mutations_sync = 1",
			table,
			formatTime(start),
			formatTime(end),
		)
		if err := s.backfillStore().Conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("clear rollup chunk %s: %w", table, err)
		}
	}

	if err := s.backfillStore().Conn.Exec(ctx, buildTraceServiceBackfillQuery(start, end)); err != nil {
		return fmt.Errorf("backfill trace service rollup: %w", err)
	}
	if err := s.backfillStore().Conn.Exec(ctx, buildTraceEndpointBackfillQuery(start, end)); err != nil {
		return fmt.Errorf("backfill trace endpoint rollup: %w", err)
	}
	if err := s.backfillStore().Conn.Exec(ctx, buildLogLevelBackfillQuery(start, end)); err != nil {
		return fmt.Errorf("backfill log level rollup: %w", err)
	}
	return nil
}

func (s *Service) backfillStore() *storage.Client {
	if s.MaintenanceStore != nil {
		return s.MaintenanceStore
	}
	return s.Storage
}
