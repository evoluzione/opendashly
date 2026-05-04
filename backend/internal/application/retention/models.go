package retention

import "time"

type RetentionSetting struct {
	ID            string
	SignalType    string
	RetentionDays uint32
	UpdatedAt     time.Time
	UpdatedBy     string
}

type CleanupJob struct {
	JobID          string
	JobType        string
	SignalType     string
	ServiceName    string
	StartedAt      time.Time
	CompletedAt    *time.Time
	Status         string
	RecordsDeleted uint64
	ErrorMessage   string
}

type CleanupRequest struct {
	SignalTypes []string
	ServiceName string
}

type CleanupResult struct {
	JobID          string
	SignalType     string
	RecordsDeleted uint64
}

const (
	RetentionModeNormal    = "normal"
	RetentionModeReduced   = "reduced"
	RetentionModeEmergency = "emergency"
)

type AdaptiveRetentionOptions struct {
	Enabled                             bool
	StepDownDays                        uint32
	MaxLevel                            int
	MinTraceRetentionDays               uint32
	MinLogRetentionDays                 uint32
	PressureCooldown                    time.Duration
	PressureMinActiveSignals            int
	PressureErrorWindow                 time.Duration
	PressureErrorThreshold              int
	PressureMemoryThresholdPercent      int
	PressureMemoryBudgetMiB             int
	PressureClickHouseDiskThresholdPerc int
}

type EmergencyRetentionState struct {
	Enabled              bool
	Mode                 string
	Level                int
	LastTransitionAt     time.Time
	LastPressureAt       *time.Time
	PressureCooldownEnds *time.Time
	LastReasons          []string
	Snapshot             PressureSnapshot
}

type PressureSnapshot struct {
	At               time.Time
	MemoryUsageMiB   uint64
	MemoryThreshold  uint64
	MemoryPercent    float64
	DiskUsedBytes    uint64
	DiskTotalBytes   uint64
	DiskUsagePercent float64
	RecentErrors     int
	Reasons          []string
	Pressure         bool
}

type SetEmergencyRequest struct {
	Enabled bool
}
