package entity

import "fmt"

type PreparedExperimentStatus string

const (
	PrepStatusStaged    PreparedExperimentStatus = "staged"
	PrepStatusRemoveBGR PreparedExperimentStatus = "removebgr"
	PrepStatusCropping  PreparedExperimentStatus = "cropping"
	PrepStatusDone      PreparedExperimentStatus = "done"
	PrepStatusFailed    PreparedExperimentStatus = "failed"
)

var validPrepStatuses = map[PreparedExperimentStatus]bool{
	PrepStatusStaged:    true,
	PrepStatusRemoveBGR: true,
	PrepStatusCropping:  true,
	PrepStatusDone:      true,
	PrepStatusFailed:    true,
}

func (s PreparedExperimentStatus) IsValid() bool {
	return validPrepStatuses[s]
}

type BGRType string

const (
	BGRFile    BGRType = "file"
	BGRAvgTail BGRType = "avgTail"
	BGRMedTail BGRType = "medTail"
)

var validBGRTypes = map[BGRType]bool{
	BGRFile:    true,
	BGRAvgTail: true,
	BGRMedTail: true,
}

func (t BGRType) IsValid() bool {
	return validBGRTypes[t]
}

type PreparedExperiment struct {
	ID           uint                     `json:"id"`
	UserID       uint                     `json:"user_id"`
	ExperimentID uint                     `json:"experiment_id"`
	CropAlt      float64                  `json:"crop_alt"`
	BGRType      BGRType                  `json:"bgr_type"`
	BGRAlt       float64                  `json:"bgr_alt,omitempty"`
	PathToData   string                   `json:"path_to_data"`
	Status       PreparedExperimentStatus `json:"status"`
	ErrorMsg     string                   `json:"error_msg,omitempty"`
}

// ValidateTransition checks if the status transition is allowed.
func (e *PreparedExperiment) ValidateTransition(newStatus PreparedExperimentStatus) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("invalid prepared experiment status: %s", newStatus)
	}
	allowed := map[PreparedExperimentStatus][]PreparedExperimentStatus{
		PrepStatusStaged:    {PrepStatusRemoveBGR, PrepStatusFailed},
		PrepStatusRemoveBGR: {PrepStatusCropping, PrepStatusFailed},
		PrepStatusCropping:  {PrepStatusDone, PrepStatusFailed},
		PrepStatusDone:      {},
		PrepStatusFailed:    {},
	}
	for _, s := range allowed[e.Status] {
		if s == newStatus {
			return nil
		}
	}
	return fmt.Errorf("invalid status transition: %s -> %s", e.Status, newStatus)
}

// ValidateBGRParams returns an error if required BGR parameters are missing.
func (e *PreparedExperiment) ValidateBGRParams() error {
	if !e.BGRType.IsValid() {
		return fmt.Errorf("invalid bgr_type: %s", e.BGRType)
	}
	if (e.BGRType == BGRAvgTail || e.BGRType == BGRMedTail) && e.BGRAlt <= 0 {
		return fmt.Errorf("bgr_alt is required and must be positive when bgr_type is %s", e.BGRType)
	}
	return nil
}
