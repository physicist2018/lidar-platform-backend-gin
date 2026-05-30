package dto

// ErrorResponse is the standard JSON error payload returned by the API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
