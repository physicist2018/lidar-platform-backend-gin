package entity

// ExperimentChart represents a cached visualization chart stored in Minio.
type ExperimentChart struct {
	ID           uint    `json:"id"`
	ExperimentID uint    `json:"experiment_id"`
	ChartType    string  `json:"chart_type"` // "image" or "profile"
	Formula      string  `json:"formula"`    // "raw", "rangecorr", "lograngecorr"
	Wavelen      float64 `json:"wavelen"`
	Polarization string  `json:"polarization"`
	IsPhoton     int8    `json:"is_photon"`      // 0=analog, 1=photon
	Glued        int8    `json:"glued"`          // 0=non-glued, 1=glued
	PathToObject string  `json:"path_to_object"` // Minio object path
}
