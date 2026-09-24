package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type Repository interface {
	GetSettings(ctx context.Context) ([]RetentionSetting, error)
	GetSettingBySignal(ctx context.Context, signalType string) (*RetentionSetting, error)
	UpdateSetting(ctx context.Context, signalType string, retentionDays uint32, updatedBy string) error
	CreateCleanupJob(ctx context.Context, job CleanupJob) error
	ListCleanupJobs(ctx context.Context, limit int) ([]CleanupJob, error)
	CountCleanupJobs(ctx context.Context) (uint64, error)
}

type Repo struct {
	Conn driver.Conn
}

func (r *Repo) EnsureSetting(ctx context.Context, signalType string, retentionDays uint32, updatedBy string) error {
	if signalType != "logs" && signalType != "traces" {
		return fmt.Errorf("unsupported signal type: %s", signalType)
	}
	query := `INSERT INTO telemetry.retention_settings (id, signal_type, retention_days, updated_at, updated_by)
	          SELECT generateUUIDv4(), ?, ?, now(), ?
	          WHERE NOT EXISTS (
	            SELECT 1 FROM telemetry.retention_settings WHERE signal_type = ?
	          )`
	if err := r.Conn.Exec(ctx, query, signalType, retentionDays, updatedBy, signalType); err != nil {
		return fmt.Errorf("ensure retention setting for %s: %w", signalType, err)
	}
	return nil
}

func (r *Repo) GetSettings(ctx context.Context) ([]RetentionSetting, error) {
	query := `SELECT id, signal_type, retention_days, updated_at, updated_by
	          FROM (
	            SELECT id, signal_type, retention_days, updated_at, updated_by
	            FROM telemetry.retention_settings
	            WHERE signal_type IN ('logs', 'traces')
	            ORDER BY updated_at DESC
	            LIMIT 1 BY signal_type
	          )
	          ORDER BY signal_type`

	rows, err := r.Conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query retention settings: %w", err)
	}
	defer rows.Close()

	var settings []RetentionSetting
	for rows.Next() {
		var s RetentionSetting
		if err := rows.Scan(&s.ID, &s.SignalType, &s.RetentionDays, &s.UpdatedAt, &s.UpdatedBy); err != nil {
			return nil, fmt.Errorf("scan retention setting: %w", err)
		}
		settings = append(settings, s)
	}

	return settings, rows.Err()
}

func (r *Repo) GetSettingBySignal(ctx context.Context, signalType string) (*RetentionSetting, error) {
	query := `SELECT id, signal_type, retention_days, updated_at, updated_by
	          FROM telemetry.retention_settings
	          WHERE signal_type = ?
	            AND signal_type IN ('logs', 'traces')
	          ORDER BY updated_at DESC
	          LIMIT 1`

	var s RetentionSetting
	row := r.Conn.QueryRow(ctx, query, signalType)
	if err := row.Scan(&s.ID, &s.SignalType, &s.RetentionDays, &s.UpdatedAt, &s.UpdatedBy); err != nil {
		return nil, fmt.Errorf("scan retention setting: %w", err)
	}

	return &s, nil
}

func (r *Repo) UpdateSetting(ctx context.Context, signalType string, retentionDays uint32, updatedBy string) error {
	query := `ALTER TABLE telemetry.retention_settings
	          UPDATE retention_days = ?, updated_at = ?, updated_by = ?
	          WHERE signal_type = ?`

	err := r.Conn.Exec(ctx, query, retentionDays, time.Now(), updatedBy, signalType)
	if err != nil {
		return fmt.Errorf("update retention setting: %w", err)
	}

	return nil
}

func (r *Repo) CreateCleanupJob(ctx context.Context, job CleanupJob) error {
	query := `INSERT INTO telemetry.cleanup_jobs
	          (job_id, job_type, signal_type, service_name, started_at, completed_at, status, records_deleted, error_message)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	err := r.Conn.Exec(ctx, query,
		job.JobID,
		job.JobType,
		job.SignalType,
		job.ServiceName,
		job.StartedAt,
		job.CompletedAt,
		job.Status,
		job.RecordsDeleted,
		job.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("create cleanup job: %w", err)
	}

	return nil
}

// visibleJobsFilter hides legacy no-op rows (old automatic runs recorded 0
// deleted records every interval) so the history shows real deletions/failures.
const visibleJobsFilter = "(records_deleted > 0 OR status = 'failed')"

func (r *Repo) ListCleanupJobs(ctx context.Context, limit int) ([]CleanupJob, error) {
	base := `SELECT job_id, job_type, signal_type, service_name, started_at, completed_at, status, records_deleted, error_message
	          FROM telemetry.cleanup_jobs
	          WHERE ` + visibleJobsFilter + `
	          ORDER BY started_at DESC`
	var rows driver.Rows
	var err error
	if limit > 0 {
		rows, err = r.Conn.Query(ctx, base+" LIMIT ?", limit)
	} else {
		rows, err = r.Conn.Query(ctx, base)
	}
	if err != nil {
		return nil, fmt.Errorf("query cleanup jobs: %w", err)
	}
	defer rows.Close()

	var jobs []CleanupJob
	for rows.Next() {
		var j CleanupJob
		if err := rows.Scan(&j.JobID, &j.JobType, &j.SignalType, &j.ServiceName, &j.StartedAt, &j.CompletedAt, &j.Status, &j.RecordsDeleted, &j.ErrorMessage); err != nil {
			return nil, fmt.Errorf("scan cleanup job: %w", err)
		}
		jobs = append(jobs, j)
	}

	return jobs, rows.Err()
}

func (r *Repo) CountCleanupJobs(ctx context.Context) (uint64, error) {
	var count uint64
	err := r.Conn.QueryRow(ctx, "SELECT count() FROM telemetry.cleanup_jobs WHERE "+visibleJobsFilter).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count cleanup jobs: %w", err)
	}
	return count, nil
}

func newJobID() string {
	return uuid.New().String()
}
