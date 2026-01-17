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
