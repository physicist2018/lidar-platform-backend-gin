package entity

import "time"

// ProcessingStatus represents the status of a processing run.
type ProcessingStatus string

const (
	ProcStatusStaged     ProcessingStatus = "staged"
	ProcStatusProcessing ProcessingStatus = "processing"
	ProcStatusDone       ProcessingStatus = "done"
	ProcStatusFailed     ProcessingStatus = "failed"
)

var validProcessingStatuses = map[ProcessingStatus]bool{
	ProcStatusStaged:     true,
	ProcStatusProcessing: true,
	ProcStatusDone:       true,
	ProcStatusFailed:     true,
}

func (s ProcessingStatus) IsValid() bool {
	return validProcessingStatuses[s]
}

// ProcessingRun represents a single processing run (algorithm execution) on an experiment.
type ProcessingRun struct {
	ID           uint
	ExperimentID uint
	UserID       uint
	Algorithm    string // "stage0", "stage1", ...
	Params       []byte // raw JSON — algorithm-specific parameters
	Status       ProcessingStatus
	ErrorMsg     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
