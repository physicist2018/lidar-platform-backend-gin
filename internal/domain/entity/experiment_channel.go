package entity

// ExperimentChannel represents a single measurement channel within an experiment,
// identified by wavelength, polarization, and photon/analog mode.
type ExperimentChannel struct {
	Wavelength   float64 `json:"wavelen"`
	Polarization string  `json:"polarization"`
	IsPhoton     int     `json:"isPhoton"` // 0 = analog, 1 = photon
	IsActive     int     `json:"isActive"` // 0 = no signal, 1 = has signal
}
