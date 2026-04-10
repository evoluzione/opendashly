package retention

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type CleanupService struct {
	Repo              Repository
	Conn              driver.Conn
	EnableCountBefore bool
}

func (s *CleanupService) CleanupByRetention(ctx context.Context) error {
	settings, err := s.Repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("get retention settings: %w", err)
	}

	failures := make([]string, 0)

	for _, setting := range settings {
		cutoffTime := time.Now().UTC().AddDate(0, 0, -int(setting.RetentionDays))
		log.Printf("retention cleanup: processing %s (retention: %d days, cutoff: %s)",
			setting.SignalType, setting.RetentionDays, cutoffTime.Format(time.RFC3339))

		jobID := uuid.New().String()
		job := CleanupJob{
			JobID:       jobID,
			JobType:     "automatic",
			SignalType:  setting.SignalType,
			ServiceName: "",
			StartedAt:   time.Now(),
			Status:      "running",
		}

		if err := s.Repo.CreateCleanupJob(ctx, job); err != nil {
			log.Printf("retention cleanup: failed to create job for %s: %v", setting.SignalType, err)
			continue
		}

		deleted, err := s.deleteOldRecords(ctx, setting.SignalType, cutoffTime, "")
		if err != nil {
			log.Printf("retention cleanup: failed to delete %s records: %v", setting.SignalType, err)
			failures = append(failures, fmt.Sprintf("%s: %v", setting.SignalType, err))
			_ = s.Repo.UpdateCleanupJob(ctx, jobID, "failed", 0, err.Error())
			continue
		}

		if err := s.Repo.UpdateCleanupJob(ctx, jobID, "completed", deleted, ""); err != nil {
			log.Printf("retention cleanup: failed to update job %s: %v", jobID, err)
		} else {
			log.Printf("retention cleanup: completed %s (deleted %d records)", setting.SignalType, deleted)
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("retention cleanup completed with %d failure(s): %s", len(failures), strings.Join(failures, "; "))
	}

	return nil
}

func (s *CleanupService) CleanupAll(ctx context.Context, signalTypes []string) ([]CleanupResult, error) {
	var results []CleanupResult
	jobID := uuid.New().String()

	for _, signalType := range signalTypes {
		job := CleanupJob{
			JobID:       jobID,
			JobType:     "manual_all",
			SignalType:  signalType,
			ServiceName: "",
			StartedAt:   time.Now(),
			Status:      "running",
		}

		if err := s.Repo.CreateCleanupJob(ctx, job); err != nil {
			return nil, fmt.Errorf("create cleanup job: %w", err)
		}

		deleted, err := s.deleteAllRecords(ctx, signalType, "")
		if err != nil {
			_ = s.Repo.UpdateCleanupJob(ctx, jobID, "failed", 0, err.Error())
			return nil, fmt.Errorf("delete %s records: %w", signalType, err)
		}

		if err := s.Repo.UpdateCleanupJob(ctx, jobID, "completed", deleted, ""); err != nil {
			log.Printf("failed to update job %s: %v", jobID, err)
		}

		results = append(results, CleanupResult{
			JobID:          jobID,
			SignalType:     signalType,
			RecordsDeleted: deleted,
		})
	}

	return results, nil
}

func (s *CleanupService) CleanupByService(ctx context.Context, signalTypes []string, serviceName string) ([]CleanupResult, error) {
	var results []CleanupResult
	jobID := uuid.New().String()

	for _, signalType := range signalTypes {
		job := CleanupJob{
			JobID:       jobID,
			JobType:     "manual_service",
			SignalType:  signalType,
			ServiceName: serviceName,
			StartedAt:   time.Now(),
			Status:      "running",
		}

		if err := s.Repo.CreateCleanupJob(ctx, job); err != nil {
			return nil, fmt.Errorf("create cleanup job: %w", err)
		}

		deleted, err := s.deleteAllRecords(ctx, signalType, serviceName)
		if err != nil {
			_ = s.Repo.UpdateCleanupJob(ctx, jobID, "failed", 0, err.Error())
			return nil, fmt.Errorf("delete %s records for service %s: %w", signalType, serviceName, err)
		}

		if err := s.Repo.UpdateCleanupJob(ctx, jobID, "completed", deleted, ""); err != nil {
			log.Printf("failed to update job %s: %v", jobID, err)
		}

		results = append(results, CleanupResult{
			JobID:          jobID,
			SignalType:     signalType,
			RecordsDeleted: deleted,
		})
	}

	return results, nil
}

func (s *CleanupService) deleteOldRecords(ctx context.Context, signalType string, cutoffTime time.Time, serviceName string) (uint64, error) {
	switch signalType {
	case "logs":
		return s.deleteLogsRecords(ctx, &cutoffTime, serviceName)
	case "traces":
		return s.deleteTracesRecords(ctx, &cutoffTime, serviceName)
	case "metrics":
		return s.deleteMetricsRecords(ctx, &cutoffTime, serviceName)
	default:
		return 0, fmt.Errorf("unknown signal type: %s", signalType)
	}
}

func (s *CleanupService) deleteAllRecords(ctx context.Context, signalType string, serviceName string) (uint64, error) {
	switch signalType {
	case "logs":
		return s.deleteLogsRecords(ctx, nil, serviceName)
	case "traces":
		return s.deleteTracesRecords(ctx, nil, serviceName)
	case "metrics":
		return s.deleteMetricsRecords(ctx, nil, serviceName)
	default:
		return 0, fmt.Errorf("unknown signal type: %s", signalType)
	}
}

func (s *CleanupService) deleteLogsRecords(ctx context.Context, cutoffTime *time.Time, serviceName string) (uint64, error) {
	var countQuery string
	var deleteQuery string

	if serviceName != "" {
		countQuery = "SELECT count() FROM telemetry.otel_logs WHERE ServiceName = ?"
		deleteQuery = "ALTER TABLE telemetry.otel_logs DELETE WHERE ServiceName = ?"
		if cutoffTime != nil {
			countQuery = "SELECT count() FROM telemetry.otel_logs WHERE Timestamp < ? AND ServiceName = ?"
			deleteQuery = "ALTER TABLE telemetry.otel_logs DELETE WHERE Timestamp < ? AND ServiceName = ?"
		}
	} else {
		countQuery = "SELECT count() FROM telemetry.otel_logs"
		deleteQuery = "ALTER TABLE telemetry.otel_logs DELETE WHERE 1=1"
		if cutoffTime != nil {
			countQuery = "SELECT count() FROM telemetry.otel_logs WHERE Timestamp < ?"
			deleteQuery = "ALTER TABLE telemetry.otel_logs DELETE WHERE Timestamp < ?"
		}
	}

	var count uint64
	args := buildDeleteArgs(cutoffTime, serviceName)
	if s.EnableCountBefore {
		row := s.Conn.QueryRow(ctx, countQuery, args...)
		if err := row.Scan(&count); err != nil {
			return 0, fmt.Errorf("count logs: %w", err)
		}
	}

	// Execute delete
	if err := s.Conn.Exec(ctx, deleteQuery, args...); err != nil {
		return 0, fmt.Errorf("delete logs: %w", err)
	}

	return count, nil
}

func (s *CleanupService) deleteTracesRecords(ctx context.Context, cutoffTime *time.Time, serviceName string) (uint64, error) {
	var countQuery string
	var deleteQuery string

	if serviceName != "" {
		countQuery = "SELECT count() FROM telemetry.otel_traces WHERE ServiceName = ?"
		deleteQuery = "ALTER TABLE telemetry.otel_traces DELETE WHERE ServiceName = ?"
		if cutoffTime != nil {
			countQuery = "SELECT count() FROM telemetry.otel_traces WHERE Timestamp < ? AND ServiceName = ?"
			deleteQuery = "ALTER TABLE telemetry.otel_traces DELETE WHERE Timestamp < ? AND ServiceName = ?"
		}
	} else {
		countQuery = "SELECT count() FROM telemetry.otel_traces"
		deleteQuery = "ALTER TABLE telemetry.otel_traces DELETE WHERE 1=1"
		if cutoffTime != nil {
			countQuery = "SELECT count() FROM telemetry.otel_traces WHERE Timestamp < ?"
			deleteQuery = "ALTER TABLE telemetry.otel_traces DELETE WHERE Timestamp < ?"
		}
	}

	var count uint64
	args := buildDeleteArgs(cutoffTime, serviceName)
	if s.EnableCountBefore {
		row := s.Conn.QueryRow(ctx, countQuery, args...)
		if err := row.Scan(&count); err != nil {
			return 0, fmt.Errorf("count traces: %w", err)
		}
	}

	// Execute delete
	if err := s.Conn.Exec(ctx, deleteQuery, args...); err != nil {
		return 0, fmt.Errorf("delete traces: %w", err)
	}

	return count, nil
}

func (s *CleanupService) deleteMetricsRecords(ctx context.Context, cutoffTime *time.Time, serviceName string) (uint64, error) {
	// Delete from both metrics tables
	countSum, err := s.deleteMetricsTable(ctx, "telemetry.otel_metrics_sum", cutoffTime, serviceName)
	if err != nil {
		return 0, err
	}

	countGauge, err := s.deleteMetricsTable(ctx, "telemetry.otel_metrics_gauge", cutoffTime, serviceName)
	if err != nil {
		return 0, err
	}

	return countSum + countGauge, nil
}

func (s *CleanupService) deleteMetricsTable(ctx context.Context, tableName string, cutoffTime *time.Time, serviceName string) (uint64, error) {
	var countQuery string
	var deleteQuery string

	if serviceName != "" {
		countQuery = fmt.Sprintf("SELECT count() FROM %s WHERE ServiceName = ?", tableName)
		deleteQuery = fmt.Sprintf("ALTER TABLE %s DELETE WHERE ServiceName = ?", tableName)
		if cutoffTime != nil {
			countQuery = fmt.Sprintf("SELECT count() FROM %s WHERE TimeUnix < ? AND ServiceName = ?", tableName)
			deleteQuery = fmt.Sprintf("ALTER TABLE %s DELETE WHERE TimeUnix < ? AND ServiceName = ?", tableName)
		}
	} else {
		countQuery = fmt.Sprintf("SELECT count() FROM %s", tableName)
		deleteQuery = fmt.Sprintf("ALTER TABLE %s DELETE WHERE 1=1", tableName)
		if cutoffTime != nil {
			countQuery = fmt.Sprintf("SELECT count() FROM %s WHERE TimeUnix < ?", tableName)
			deleteQuery = fmt.Sprintf("ALTER TABLE %s DELETE WHERE TimeUnix < ?", tableName)
		}
	}

	var count uint64
	args := buildDeleteArgs(cutoffTime, serviceName)
	if s.EnableCountBefore {
		row := s.Conn.QueryRow(ctx, countQuery, args...)
		if err := row.Scan(&count); err != nil {
			return 0, fmt.Errorf("count %s: %w", tableName, err)
		}
	}

	// Execute delete
	if err := s.Conn.Exec(ctx, deleteQuery, args...); err != nil {
		return 0, fmt.Errorf("delete %s: %w", tableName, err)
	}

	return count, nil
}

func buildDeleteArgs(cutoffTime *time.Time, serviceName string) []interface{} {
	if cutoffTime != nil && serviceName != "" {
		return []interface{}{*cutoffTime, serviceName}
	}
	if cutoffTime != nil {
		return []interface{}{*cutoffTime}
	}
	if serviceName != "" {
		return []interface{}{serviceName}
	}
	return nil
}
