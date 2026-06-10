package dto

type PrepareExperimentBody struct {
	CropAlt float64 `json:"crop_alt" validate:"required,min=0"`
	BGRType string  `json:"bgr_type" validate:"required,oneof=file avgTail medTail"`
	BGRAlt  float64 `json:"bgr_alt" validate:"omitempty,min=0"`
}

type PreparedExperimentResponse struct {
	ID           uint    `json:"id"`
	UserID       uint    `json:"user_id"`
	ExperimentID uint    `json:"experiment_id"`
	CropAlt      float64 `json:"crop_alt"`
	BGRType      string  `json:"bgr_type"`
	BGRAlt       float64 `json:"bgr_alt,omitempty"`
	PathToData   string  `json:"path_to_data"`
	Status       string  `json:"status"`
	ErrorMsg     string  `json:"error_msg,omitempty"`
}
