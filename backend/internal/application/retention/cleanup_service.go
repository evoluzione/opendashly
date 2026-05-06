package retention

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type CleanupService struct {
	Repo              Repository
	Conn              driver.Conn
	EnableCountBefore bool
	AdaptiveOptions   AdaptiveRetentionOptions
	PressureMonitor   *PressureMonitor

	mu                sync.Mutex
	emergencyEnabled  bool
	emergencyLevel    int
	lastTransitionAt  time.Time
	lastPressureAt    *time.Time
	pressureUntil     *time.Time
	lastPressureCause []string
}

func (s *CleanupService) EnsureDefaults() {
	if s.AdaptiveOptions.StepDownDays == 0 {
		s.AdaptiveOptions.StepDownDays = 2
	}
	if s.AdaptiveOptions.MaxLevel <= 0 {
		s.AdaptiveOptions.MaxLevel = 2
	}
	if s.AdaptiveOptions.MinTraceRetentionDays == 0 {
		s.AdaptiveOptions.MinTraceRetentionDays = 1
	}
	if s.AdaptiveOptions.MinLogRetentionDays == 0 {
		s.AdaptiveOptions.MinLogRetentionDays = 1
	}
	if s.AdaptiveOptions.PressureCooldown <= 0 {
		s.AdaptiveOptions.PressureCooldown = 5 * time.Minute
	}
	if s.AdaptiveOptions.PressureMinActiveSignals <= 0 {
		s.AdaptiveOptions.PressureMinActiveSignals = 2
	}
	if s.AdaptiveOptions.PressureErrorWindow <= 0 {
		s.AdaptiveOptions.PressureErrorWindow = 5 * time.Minute
	}
	if s.AdaptiveOptions.PressureErrorThreshold <= 0 {
		s.AdaptiveOptions.PressureErrorThreshold = 4
	}
	if s.AdaptiveOptions.PressureMemoryThresholdPercent <= 0 {
		s.AdaptiveOptions.PressureMemoryThresholdPercent = 85
	}
	if s.AdaptiveOptions.PressureClickHouseDiskThresholdPerc <= 0 {
		s.AdaptiveOptions.PressureClickHouseDiskThresholdPerc = 90
	}

	s.mu.Lock()
	if s.lastTransitionAt.IsZero() {
		s.lastTransitionAt = time.Now().UTC()
	}
	s.emergencyEnabled = s.AdaptiveOptions.Enabled
	s.mu.Unlock()
}

func (s *CleanupService) ObserveQueryError(msg string) {
	if s.PressureMonitor == nil {
		return
	}
	s.PressureMonitor.ObserveRecoverableError(msg)
}

func (s *CleanupService) SetEmergencyEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emergencyEnabled = enabled
	s.lastTransitionAt = time.Now().UTC()
	if !enabled {
		s.emergencyLevel = 0
		s.lastPressureCause = nil
		s.lastPressureAt = nil
		s.pressureUntil = nil
	}
}

func (s *CleanupService) EmergencyState(ctx context.Context) EmergencyRetentionState {
	snapshot := PressureSnapshot{}
	if s.PressureMonitor != nil {
		snapshot = s.PressureMonitor.Snapshot(ctx)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	state := EmergencyRetentionState{
		Enabled:          s.emergencyEnabled,
		Mode:             modeForLevel(s.emergencyLevel),
		Level:            s.emergencyLevel,
		LastTransitionAt: s.lastTransitionAt,
		LastReasons:      append([]string(nil), s.lastPressureCause...),
		Snapshot:         snapshot,
	}
	if s.lastPressureAt != nil {
		t := *s.lastPressureAt
		state.LastPressureAt = &t
	}
	if s.pressureUntil != nil {
		t := *s.pressureUntil
		state.PressureCooldownEnds = &t
	}
	return state
}

func (s *CleanupService) CleanupByRetention(ctx context.Context) error {
	s.EnsureDefaults()
	effectiveLevel, mode, snapshot := s.currentLevelAndMode(ctx)
	if mode != RetentionModeNormal {
		log.Printf("retention cleanup: adaptive mode=%s level=%d reasons=%v", mode, effectiveLevel, snapshot.Reasons)
	}

	settings, err := s.Repo.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("get retention settings: %w", err)
	}

	failures := make([]string, 0)

	for _, setting := range settings {
		effectiveDays := s.effectiveRetentionDays(setting, effectiveLevel)
		cutoffTime := time.Now().UTC().AddDate(0, 0, -int(effectiveDays))
		log.Printf("retention cleanup: processing %s (retention: %d days, cutoff: %s)",
			setting.SignalType, effectiveDays, cutoffTime.Format(time.RFC3339))

		jobID := uuid.New().String()
		jobType := "automatic"
		if mode != RetentionModeNormal {
			jobType = "automatic_" + mode
		}
		job := CleanupJob{
			JobID:       jobID,
			JobType:     jobType,
			SignalType:  setting.SignalType,
			ServiceName: "",
			StartedAt:   time.Now(),
			Status:      "running",
		}

		if err := s.Repo.CreateCleanupJob(ctx, job); err != nil {
			log.Printf("retention cleanup: failed to create job for %s: %v", setting.SignalType, err)
			continue
		}

		deleted, err := s.deleteOldRecords(ctx, setting.SignalType, cutoffTime, "", s.EnableCountBefore)
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

func (s *CleanupService) effectiveRetentionDays(setting RetentionSetting, level int) uint32 {
	base := setting.RetentionDays
	if level <= 0 {
		return clampMinRetention(base, setting.SignalType, s.AdaptiveOptions)
	}

	reduction := uint32(level) * s.AdaptiveOptions.StepDownDays
	if reduction >= base {
		return minRetentionForSignal(setting.SignalType, s.AdaptiveOptions)
	}
	adjusted := base - reduction
	return clampMinRetention(adjusted, setting.SignalType, s.AdaptiveOptions)
}

func (s *CleanupService) currentLevelAndMode(ctx context.Context) (int, string, PressureSnapshot) {
	if s.PressureMonitor == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.emergencyLevel, modeForLevel(s.emergencyLevel), PressureSnapshot{}
	}

	snapshot := s.PressureMonitor.Snapshot(ctx)
	now := snapshot.At

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.emergencyEnabled {
		s.emergencyLevel = 0
		s.lastPressureCause = nil
		return 0, RetentionModeNormal, snapshot
	}

	if snapshot.Pressure {
		if s.emergencyLevel < s.AdaptiveOptions.MaxLevel {
			s.emergencyLevel++
			s.lastTransitionAt = now
		}
		s.lastPressureCause = append([]string(nil), snapshot.Reasons...)
		s.lastPressureAt = &now
		cooldownEnd := now.Add(s.AdaptiveOptions.PressureCooldown)
		s.pressureUntil = &cooldownEnd
		return s.emergencyLevel, modeForLevel(s.emergencyLevel), snapshot
	}

	if s.pressureUntil != nil && now.Before(*s.pressureUntil) {
		return s.emergencyLevel, modeForLevel(s.emergencyLevel), snapshot
	}

	if s.emergencyLevel > 0 {
		s.emergencyLevel--
		s.lastTransitionAt = now
	}
	if s.emergencyLevel == 0 {
		s.lastPressureCause = nil
		s.lastPressureAt = nil
		s.pressureUntil = nil
	}
	return s.emergencyLevel, modeForLevel(s.emergencyLevel), snapshot
}

func modeForLevel(level int) string {
	if level <= 0 {
		return RetentionModeNormal
	}
	if level == 1 {
		return RetentionModeReduced
	}
	return RetentionModeEmergency
}

func minRetentionForSignal(signalType string, opts AdaptiveRetentionOptions) uint32 {
	if signalType == "traces" {
		if opts.MinTraceRetentionDays > 0 {
			return opts.MinTraceRetentionDays
		}
		return 1
	}
	if opts.MinLogRetentionDays > 0 {
		return opts.MinLogRetentionDays
	}
	return 1
}

func clampMinRetention(days uint32, signalType string, opts AdaptiveRetentionOptions) uint32 {
	minDays := minRetentionForSignal(signalType, opts)
	if days < minDays {
		return minDays
	}
	return days
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

		deleted, err := s.deleteAllRecords(ctx, signalType, "", true)
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

		deleted, err := s.deleteAllRecords(ctx, signalType, serviceName, true)
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

func (s *CleanupService) deleteOldRecords(ctx context.Context, signalType string, cutoffTime time.Time, serviceName string, countBefore bool) (uint64, error) {
	switch signalType {
	case "logs":
		return s.deleteLogsRecords(ctx, &cutoffTime, serviceName, countBefore)
	case "traces":
		return s.deleteTracesRecords(ctx, &cutoffTime, serviceName, countBefore)
	default:
		return 0, fmt.Errorf("unknown signal type: %s", signalType)
	}
}

func (s *CleanupService) deleteAllRecords(ctx context.Context, signalType string, serviceName string, countBefore bool) (uint64, error) {
	switch signalType {
	case "logs":
		return s.deleteLogsRecords(ctx, nil, serviceName, countBefore)
	case "traces":
		return s.deleteTracesRecords(ctx, nil, serviceName, countBefore)
	default:
		return 0, fmt.Errorf("unknown signal type: %s", signalType)
	}
}

func (s *CleanupService) deleteLogsRecords(ctx context.Context, cutoffTime *time.Time, serviceName string, countBefore bool) (uint64, error) {
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
	if countBefore {
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

func (s *CleanupService) deleteTracesRecords(ctx context.Context, cutoffTime *time.Time, serviceName string, countBefore bool) (uint64, error) {
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
	if countBefore {
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
