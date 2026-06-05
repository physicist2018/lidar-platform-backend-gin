package dto

// TaskStatusResponse is returned by the task polling endpoint.
type TaskStatusResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`          // pending, processing, done, failed
	URL    string `json:"url,omitempty"`   // presigned Minio URL (when done)
	Error  string `json:"error,omitempty"` // error message (when failed)
}
