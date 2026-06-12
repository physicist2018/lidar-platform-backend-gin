package queue

const (
	// TypePrepare is the async task for experiment preprocessing.
	TypePrepare = "task:prepare"
	// TypeGlue is the async task for channel gluing.
	TypeGlue = "task:glue"
	// TypeVisualize is the async task for chart generation.
	TypeVisualize = "task:visualize"
	// TypeProcess is the async task for algorithm processing (stage0, stage1, ...).
	TypeProcess = "task:process"
)

// PreparePayload is the task payload for experiment preprocessing.
type PreparePayload struct {
	PrepID       uint    `json:"prep_id"`
	ExperimentID uint    `json:"experiment_id"`
	CropAlt      float64 `json:"crop_alt"`
	BGRType      string  `json:"bgr_type"`
	BGRAlt       float64 `json:"bgr_alt"`
	PathToData   string  `json:"path_to_data"`
}

// GluePayload is the task payload for channel gluing.
type GluePayload struct {
	PrepID       uint      `json:"prep_id"`
	ExperimentID uint      `json:"experiment_id"`
	PathToData   string    `json:"path_to_data"`
	Wavelengths  []float64 `json:"wavelengths"`
	Polarization string    `json:"polarization"`
	H1           float64   `json:"h1"`
	H2           float64   `json:"h2"`
}

// ProcessPayload is the task payload for algorithm processing.
type ProcessPayload struct {
	ProcID       uint   `json:"proc_id"`
	ExperimentID uint   `json:"experiment_id"`
	Algorithm    string `json:"algorithm"`
}

// VisualizePayload is the task payload for chart generation.
type VisualizePayload struct {
	PrepID       uint    `json:"prep_id"`
	Wavelen      float64 `json:"wavelen"`
	IsPhoton     int8    `json:"is_photon"`
	Polarization string  `json:"polarization"`
	VizType      string  `json:"viz_type"`
	OutputType   string  `json:"output_type"`
	Formula      string  `json:"formula"`
	Regenerate   bool    `json:"regenerate"`
	Glued        int8    `json:"glued"`
}
