package entity

// ProcessedSignal holds the processed signal for a single original profile.
type ProcessedSignal struct {
	ID                uint
	ProcessingRunID   uint      `json:"processing_run_id"`
	OriginalProfileID uint      `json:"original_profile_id"`
	Wavelength        float64   `json:"wavelength"`
	Polarization      string    `json:"polarization"`
	IsPhoton          bool      `json:"is_photon"`
	Signal            []float64 `json:"signal"`
}
