package dto

// VisualizeChartResponse is returned by the visualize endpoint with the task ID for polling.
type VisualizeChartResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"` // "accepted" — task was enqueued
}
