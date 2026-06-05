package dto

// VisualizePreparedExperimentURI binds path parameters for the visualize endpoint.
type VisualizePreparedExperimentURI struct {
	ID uint `uri:"id" binding:"required,min=1"`
}

// VisualizePreparedExperimentQuery binds all query parameters for the visualize endpoint.
type VisualizePreparedExperimentQuery struct {
	Wavelen      float64 `form:"wavelen" binding:"required"`
	Photon       int8    `form:"photon"` // 0=analog (default), 1=photon; ignored when glued=1
	Polarization string  `form:"polarization"`
	Action       string  `form:"action" binding:"required,oneof=image profile"`
	Glued        int8    `form:"glued"`   // 0 = non-glued (default), 1 = glued profiles only
	Type         string  `form:"type"`    // png (default), svg, or json
	Formula      string  `form:"formula"` // raw, rangecorr, lograngecorr
	Regenerate   bool    `form:"regenerate"`
}

// VisualizeChartResponse is returned by the visualize endpoint with the chart URL.
type VisualizeChartResponse struct {
	URL string `json:"url"`
}
