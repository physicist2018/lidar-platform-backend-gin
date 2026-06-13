package dto

// Stage0DataResponse is the response body for GET /results/{stage}/data.
type Stage0DataResponse struct {
	Distance []float64   `json:"distance"` // metres: i * binWidth
	Data     [][]float64 `json:"data"`     // 2D array [profileIndex][sampleIndex]
	Time     []string    `json:"time"`     // ISO 8601 start_time of each profile's file
}
