package entity

// Stage0Params is the algorithm-specific parameters for stage0 processing.
type Stage0Params struct {
	Background BackgroundParams `json:"background"`
	Glue       []GlueParam      `json:"glue"`
}

// BackgroundParams defines how background subtraction is performed.
type BackgroundParams struct {
	Type    string  `json:"type"`     // "file", "avgtail", "medtail"
	BgrFrom float64 `json:"bgr_from"` // altitude (in meters) for tail-based statistics
}

// GlueParam defines parameters for gluing analog and digital channels.
type GlueParam struct {
	Wavelength   float64 `json:"wavelength"`   // e.g. 532, 355
	Polarization string  `json:"polarization"` // "p", "s", "o"
	R0           float64 `json:"r0"`           // start altitude for overlap region (meters)
	R1           float64 `json:"r1"`           // end altitude for overlap region (meters)
	ScaleTo      string  `json:"scale_to"`     // "analog" or "digital" — which channel to scale to
}
