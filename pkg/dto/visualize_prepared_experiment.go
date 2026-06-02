package dto

// VisualizePreparedExperimentURI binds path parameters for the visualize endpoint.
type VisualizePreparedExperimentURI struct {
	ID           uint    `uri:"id" binding:"required,min=1"`
	Wavelen      float64 `uri:"wavelen" binding:"required"`
	Photon       int8    `uri:"photon" binding:"required"`
	Polarization string  `uri:"polarization"`
	Action       string  `uri:"action" binding:"required,oneof=image profile"`
}

// VisualizeTypeQuery binds the type, formula and regenerate query parameters.
type VisualizeTypeQuery struct {
	Type       string `form:"type"`
	Formula    string `form:"formula"`
	Regenerate bool   `form:"regenerate"`
}

// VisualizeChartResponse is returned by the visualize endpoint with the chart URL.
type VisualizeChartResponse struct {
	URL string `json:"url"`
}
