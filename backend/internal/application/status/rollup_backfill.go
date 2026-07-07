package status

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"opendashly/backend/internal/infrastructure/config"
	"opendashly/backend/internal/infrastructure/storage"
)

const statusRollupBackfillName = "status_v1"

func (s *Service) StartRollupBackfill(ctx context.Context, hours int) {
	store := s.backfillStore()
	if store == nil || store.Conn == nil {
		return
	}
	if hours <= 0 {
		hours = 24
	}

	go func() {
		if err := s.runStatusRollupBackfill(ctx, hours); err != nil {
			log.Printf("status.rollup_backfill: stopped: %v", err)
		}
	}()
}

func (s *Service) backfillStore() *storage.Client {
	if s.MaintenanceStorage != nil {
		return s.MaintenanceStorage
	}
	return s.Storage
}

func (s *Service) runStatusRollupBackfill(ctx context.Context, hours int) error {
	upper := time.Now().UTC().Truncate(time.Minute)
	lower := upper.Add(-time.Duration(hours) * time.Hour).Truncate(time.Minute)
	if !lower.Before(upper) {
		return nil
	}

	chunkSize := 15 * time.Minute
	for start := lower; start.Before(upper); start = start.Add(chunkSize) {
		end := start.Add(chunkSize)
		if end.After(upper) {
			end = upper
		}
		done, err := s.statusBackfillChunkCompleted(ctx, start, end)
		if err != nil {
			return err
		}
		if done {
			continue
		}
		if err := s.statusBackfillChunk(ctx, start, end); err != nil {
			if isStatusPressureError(err) {
				return err
			}
			return err
		}
		if err := s.markStatusBackfillChunkCompleted(ctx, start, end); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) statusBackfillChunkCompleted(ctx context.Context, start, end time.Time) (bool, error) {
	var count uint64
	err := s.backfillStore().Conn.QueryRow(ctx, `
		SELECT count()
		FROM telemetry.status_rollup_backfill_chunks
		WHERE rollup = ? AND chunk_start = ? AND chunk_end = ?
	`, statusRollupBackfillName, start, end).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("read status backfill marker: %w", err)
	}
	return count > 0, nil
}

func (s *Service) markStatusBackfillChunkCompleted(ctx context.Context, start, end time.Time) error {
	if err := s.backfillStore().Conn.Exec(ctx, `
		INSERT INTO telemetry.status_rollup_backfill_chunks
			(rollup, chunk_start, chunk_end, completed_at)
		VALUES (?, ?, ?, ?)
	`, statusRollupBackfillName, start, end, time.Now().UTC()); err != nil {
		return fmt.Errorf("mark status backfill marker: %w", err)
	}
	return nil
}

func (s *Service) statusBackfillChunk(ctx context.Context, start, end time.Time) error {
	log.Printf("status.rollup_backfill: chunk start=%s end=%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	for _, table := range []string{"telemetry.status_logs_1m", "telemetry.status_traces_1m"} {
		stmt := fmt.Sprintf(
			"ALTER TABLE %s DELETE WHERE time_bucket >= '%s' AND time_bucket < '%s' SETTINGS mutations_sync = 1",
			table,
			formatStatusTime(start),
			formatStatusTime(end),
		)
		if err := s.backfillStore().Conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("clear status rollup chunk %s: %w", table, err)
		}
	}

	if err := s.backfillStore().Conn.Exec(ctx, buildStatusLogsBackfillQuery(start, end)); err != nil {
		return fmt.Errorf("backfill status logs rollup: %w", err)
	}
	if err := s.backfillStore().Conn.Exec(ctx, buildStatusTracesBackfillQuery(start, end)); err != nil {
		return fmt.Errorf("backfill status traces rollup: %w", err)
	}
	return nil
}

func buildStatusLogsBackfillQuery(start, end time.Time) string {
	return fmt.Sprintf(`
		INSERT INTO telemetry.status_logs_1m
		SELECT
			toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
			count() AS total
		FROM telemetry.otel_logs
		WHERE Timestamp >= '%s' AND Timestamp < '%s'
		GROUP BY time_bucket
		SETTINGS max_execution_time = %d, max_threads = 1, max_memory_usage = 67108864,
			max_bytes_before_external_group_by = 16777216
	`, formatStatusTime(start), formatStatusTime(end), config.Auto().StatusRollupBackfillMaxExecSec)
}

func buildStatusTracesBackfillQuery(start, end time.Time) string {
	return fmt.Sprintf(`
		INSERT INTO telemetry.status_traces_1m
		SELECT
			toStartOfMinute(toDateTime(Timestamp)) AS time_bucket,
			count() AS total
		FROM telemetry.otel_traces
		WHERE Timestamp >= '%s' AND Timestamp < '%s'
		GROUP BY time_bucket
		SETTINGS max_execution_time = %d, max_threads = 1, max_memory_usage = 67108864,
			max_bytes_before_external_group_by = 16777216
	`, formatStatusTime(start), formatStatusTime(end), config.Auto().StatusRollupBackfillMaxExecSec)
}

func formatStatusTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

func isStatusPressureError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "memory limit exceeded") || strings.Contains(lower, "overcommittracker")
}
