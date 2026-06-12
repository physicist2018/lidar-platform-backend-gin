package dto

import "encoding/json"

// ProcessExperimentBody is the request body for POST /experiments/{id}/process.
type ProcessExperimentBody struct {
	Algorithm string          `json:"algorithm" validate:"required"`
	Params    json.RawMessage `json:"params" validate:"required"`
}

// ProcessingRunResponse is the response for a processing run.
type ProcessingRunResponse struct {
	ID           uint            `json:"id"`
	ExperimentID uint            `json:"experiment_id"`
	Algorithm    string          `json:"algorithm"`
	Params       json.RawMessage `json:"params"`
	Status       string          `json:"status"`
	ErrorMsg     string          `json:"error_msg,omitempty"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}
