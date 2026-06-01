package dto

// VisualizePreparedExperimentURI binds path parameters for the visualize endpoint.
type VisualizePreparedExperimentURI struct {
	ID           uint    `uri:"id" binding:"required,min=1"`
	Wavelen      float64 `uri:"wavelen" binding:"required"`
	Photon       bool    `uri:"photon"`
	Polarization string  `uri:"polarization"`
	Action       string  `uri:"action" binding:"required,oneof=image profile"`
}

// VisualizeTypeQuery binds the type query parameter.
type VisualizeTypeQuery struct {
	Type string `form:"type"`
}
