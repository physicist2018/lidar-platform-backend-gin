package entity

import (
	"fmt"
	"time"
)

type ExperimentStatus string

const (
	StatusStaged    ExperimentStatus = "staged"
	StatusUploading ExperimentStatus = "uploading"
	StatusDone      ExperimentStatus = "done"
	StatusFailed    ExperimentStatus = "failed"
)

var validStatuses = map[ExperimentStatus]bool{
	StatusStaged:    true,
	StatusUploading: true,
	StatusDone:      true,
	StatusFailed:    true,
}

func (s ExperimentStatus) IsValid() bool {
	return validStatuses[s]
}

type Experiment struct {
	ID                   uint             `json:"id"`
	UserID               uint             `json:"user_id"`
	Title                string           `json:"title"`
	Comments             string           `json:"comments"`
	MeasurementStartTime *time.Time       `json:"measurement_start_time,omitempty"`
	MeasurementStopTime  *time.Time       `json:"measurement_stop_time,omitempty"`
	LicelZipPath         string           `json:"licel_zip_path"`
	LicelBgrPath         string           `json:"licel_bgr_path"`
	MeteoFilePath        string           `json:"meteo_file_path"`
	Status               ExperimentStatus `json:"status"`
	ErrorMsg             string           `json:"error_msg,omitempty"`
	CreatedAt            time.Time        `json:"created_at"`
	UpdatedAt            time.Time        `json:"updated_at"`
}

// ValidateTransition checks if the status transition is allowed.
func (e *Experiment) ValidateTransition(newStatus ExperimentStatus) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("invalid experiment status: %s", newStatus)
	}
	allowed := map[ExperimentStatus][]ExperimentStatus{
		StatusStaged:    {StatusUploading, StatusFailed},
		StatusUploading: {StatusDone, StatusFailed},
		StatusDone:      {},
		StatusFailed:    {},
	}
	for _, s := range allowed[e.Status] {
		if s == newStatus {
			return nil
		}
	}
	return fmt.Errorf("invalid status transition: %s -> %s", e.Status, newStatus)
}
