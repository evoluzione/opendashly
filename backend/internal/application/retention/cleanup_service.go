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
	Repo            Repository
	Conn            driver.Conn
	AdaptiveOptions AdaptiveRetentionOptions
	PressureMonitor *PressureMonitor

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

		jobType := "automatic"
		if mode != RetentionModeNormal {
			jobType = "automatic_" + mode
		}
		job := CleanupJob{
			JobID:      uuid.New().String(),
			JobType:    jobType,
			SignalType: setting.SignalType,
			StartedAt:  time.Now(),
		}

		deleted, err := s.dropPartitionsBefore(ctx, setting.SignalType, cutoffTime)
		if err != nil {
			log.Printf("retention cleanup: failed to delete %s records: %v", setting.SignalType, err)
			failures = append(failures, fmt.Sprintf("%s: %v", setting.SignalType, err))
			s.recordJob(ctx, job, deleted, err)
			continue
		}
		// No-op runs happen every interval; recording them would bury the real
		// deletions in the history.
		if deleted > 0 {
			s.recordJob(ctx, job, deleted, nil)
		}
		log.Printf("retention cleanup: completed %s (deleted %d records)", setting.SignalType, deleted)
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
		}

		deleted, err := s.deleteAllRecords(ctx, signalType, "", true)
		s.recordJob(ctx, job, deleted, err)
		if err != nil {
			return nil, fmt.Errorf("delete %s records: %w", signalType, err)
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
		}

		deleted, err := s.deleteAllRecords(ctx, signalType, serviceName, true)
		s.recordJob(ctx, job, deleted, err)
		if err != nil {
			return nil, fmt.Errorf("delete %s records for service %s: %w", signalType, serviceName, err)
		}

		results = append(results, CleanupResult{
			JobID:          jobID,
			SignalType:     signalType,
			RecordsDeleted: deleted,
		})
	}

	return results, nil
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

func (s *CleanupService) recordJob(ctx context.Context, job CleanupJob, deleted uint64, err error) {
	now := time.Now()
	job.CompletedAt = &now
	job.RecordsDeleted = deleted
	job.Status = "completed"
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = err.Error()
	}
	if rerr := s.Repo.CreateCleanupJob(ctx, job); rerr != nil {
		log.Printf("retention cleanup: failed to record %s job for %s: %v", job.JobType, job.SignalType, rerr)
	}
}

// Tables are partitioned by day, so retention drops whole partitions: unlike an
// ALTER DELETE mutation (which rewrites parts and needs free disk), a drop is
// instant and frees space even on a full disk. trace_id_ts has no TTL.
var partitionTables = map[string]string{
	"otel_logs":               "logs",
	"otel_traces":             "traces",
	"otel_traces_trace_id_ts": "traces",
}

type partition struct {
	Table string
	ID    string // YYYYMMDD
	Rows  uint64
	Bytes uint64
}

func (p partition) signal() string { return partitionTables[p.Table] }

// counted reports whether the partition rows are user-visible records (the
// trace_id_ts index rows are not).
func (p partition) counted() bool { return p.Table != "otel_traces_trace_id_ts" }

func (s *CleanupService) listPartitions(ctx context.Context) ([]partition, error) {
	rows, err := s.Conn.Query(ctx, `SELECT table, partition_id, sum(rows), sum(bytes_on_disk)
		FROM system.parts
		WHERE database = 'telemetry' AND active
		  AND table IN ('otel_logs', 'otel_traces', 'otel_traces_trace_id_ts')
		GROUP BY table, partition_id
		ORDER BY partition_id, table`)
	if err != nil {
		return nil, fmt.Errorf("list partitions: %w", err)
	}
	defer rows.Close()
	var parts []partition
	for rows.Next() {
		var p partition
		if err := rows.Scan(&p.Table, &p.ID, &p.Rows, &p.Bytes); err != nil {
			return nil, fmt.Errorf("scan partition: %w", err)
		}
		if len(p.ID) == 8 {
			parts = append(parts, p)
		}
	}
	return parts, rows.Err()
}

func (s *CleanupService) dropPartitions(ctx context.Context, parts []partition) (map[string]uint64, error) {
	deleted := map[string]uint64{}
	for _, p := range parts {
		if err := s.Conn.Exec(ctx, "ALTER TABLE telemetry."+p.Table+" DROP PARTITION ID ?", p.ID); err != nil {
			return deleted, fmt.Errorf("drop partition %s of %s: %w", p.ID, p.Table, err)
		}
		if p.counted() {
			deleted[p.signal()] += p.Rows
		}
	}
	return deleted, nil
}

func (s *CleanupService) dropPartitionsBefore(ctx context.Context, signalType string, cutoff time.Time) (uint64, error) {
	parts, err := s.listPartitions(ctx)
	if err != nil {
		return 0, err
	}
	deleted, err := s.dropPartitions(ctx, partitionsBefore(parts, signalType, cutoff.UTC().Format("20060102")))
	return deleted[signalType], err
}

// partitionsBefore returns the signal's day partitions fully older than the
// cutoff day.
func partitionsBefore(parts []partition, signalType, cutoffID string) []partition {
	var out []partition
	for _, p := range parts {
		if p.signal() == signalType && p.ID < cutoffID {
			out = append(out, p)
		}
	}
	return out
}

// partitionsToFree picks the oldest partitions (parts must be sorted by ID)
// until their size covers toFree, never touching a table's newest partition.
func partitionsToFree(parts []partition, toFree uint64) []partition {
	newest := map[string]string{}
	for _, p := range parts {
		if p.ID > newest[p.Table] {
			newest[p.Table] = p.ID
		}
	}
	var out []partition
	var freed uint64
	for _, p := range parts {
		if freed >= toFree {
			break
		}
		if p.ID == newest[p.Table] {
			continue
		}
		out = append(out, p)
		freed += p.Bytes
	}
	return out
}

// PurgeForDisk drops the oldest telemetry partitions when the ClickHouse disk
// crosses the pressure threshold, regardless of retention settings: a full disk
// takes down ingestion, queries and the host itself.
func (s *CleanupService) PurgeForDisk(ctx context.Context) error {
	if s.PressureMonitor == nil {
		return nil
	}
	used, total, err := s.PressureMonitor.queryDiskStats(ctx)
	if err != nil || total == 0 {
		return err
	}
	// Dropped parts stay on disk until ClickHouse removes them (~8 min); count
	// them as already freed so consecutive checks do not over-drop.
	var pending uint64
	if err := s.Conn.QueryRow(ctx, "SELECT sum(bytes_on_disk) FROM system.parts WHERE NOT active").Scan(&pending); err != nil {
		return fmt.Errorf("query inactive parts: %w", err)
	}
	if pending < used {
		used -= pending
	} else {
		used = 0
	}
	threshold := float64(s.AdaptiveOptions.PressureClickHouseDiskThresholdPerc) / 100
	if float64(used) < threshold*float64(total) {
		return nil
	}
	// ponytail: fixed 10-point hysteresis below the threshold; make it
	// configurable if hosts need a different headroom.
	target := uint64(max(threshold-0.10, 0) * float64(total))

	parts, err := s.listPartitions(ctx)
	if err != nil {
		return err
	}
	victims := partitionsToFree(parts, used-target)
	if len(victims) == 0 {
		log.Printf("retention disk purge: disk at %.1f%% but only current partitions are left", float64(used)/float64(total)*100)
		return nil
	}
	started := time.Now()
	deleted, dropErr := s.dropPartitions(ctx, victims)
	log.Printf("retention disk purge: disk at %.1f%%, dropped %d partitions (oldest %s)", float64(used)/float64(total)*100, len(victims), victims[0].ID)
	for _, signalType := range []string{"logs", "traces"} {
		if deleted[signalType] == 0 && dropErr == nil {
			continue
		}
		job := CleanupJob{JobID: uuid.New().String(), JobType: "automatic_disk", SignalType: signalType, StartedAt: started}
		s.recordJob(ctx, job, deleted[signalType], dropErr)
	}
	return dropErr
}
