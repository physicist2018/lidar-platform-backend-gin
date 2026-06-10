package dto

// VisualizePreparedExperimentURI binds path parameters for the visualize endpoint.
type VisualizePreparedExperimentURI struct {
	ID uint `param:"id"`
}

// VisualizePreparedExperimentQuery binds all query parameters for the visualize endpoint.
type VisualizePreparedExperimentQuery struct {
	Wavelen      float64 `query:"wavelen"   validate:"required"`
	Photon       int8    `query:"photon"` // 0=analog (default), 1=photon; ignored when glued=1
	Polarization string  `query:"polarization"`
	Action       string  `query:"action"   validate:"required,oneof=image profile"`
	Glued        int8    `query:"glued"`   // 0 = non-glued (default), 1 = glued profiles only
	Type         string  `query:"type"`    // png (default), svg, or json
	Formula      string  `query:"formula"` // raw, rangecorr, lograngecorr
	Regenerate   bool    `query:"regenerate"`
}

// VisualizeChartResponse is now defined in visualize_chart_response.go
