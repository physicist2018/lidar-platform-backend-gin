package usecase

import "context"

// AsyncTaskInfo holds the result of an async task submission.
type AsyncTaskInfo struct {
	TaskID string `json:"task_id"`
}

// VisualizePreparedExperimentUseCase enqueues a visualization task and returns the task ID for polling.
type VisualizePreparedExperimentUseCase interface {
	Execute(
		ctx context.Context,
		prepID uint,
		wavelen float64,
		isPhoton int8,
		polarization string,
		vizType string,
		outputType string,
		formula string,
		regenerate bool,
		glued int8,
	) (*AsyncTaskInfo, error) // returns task ID for polling
}
