package dto

type PrepareExperimentBody struct {
	CropAlt float64 `json:"crop_alt" binding:"required,min=0"`
	BGRType string  `json:"bgr_type" binding:"required,oneof=file avgTail medTail"`
	BGRAlt  float64 `json:"bgr_alt" binding:"omitempty,min=0"`
}

type PreparedExperimentResponse struct {
	ID           uint    `json:"id"`
	ExperimentID uint    `json:"experiment_id"`
	CropAlt      float64 `json:"crop_alt"`
	BGRType      string  `json:"bgr_type"`
	BGRAlt       float64 `json:"bgr_alt,omitempty"`
	PathToData   string  `json:"path_to_data"`
	Status       string  `json:"status"`
	ErrorMsg     string  `json:"error_msg,omitempty"`
}
