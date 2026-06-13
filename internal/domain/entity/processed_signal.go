package entity

// ProcessedSignal holds the processed signal for a single original profile.

import "time"

type ProcessedSignal struct {
	ID                uint       `json:"id"`
	ProcessingRunID   uint       `json:"processing_run_id"`
	OriginalProfileID uint       `json:"original_profile_id"`
	FileID            uint       `json:"file_id"`
	FileStartTime     *time.Time `json:"-"` // populated only on read with join — not persisted
	Wavelength        float64    `json:"wavelength"`
	Polarization      string     `json:"polarization"`
	IsPhoton          bool       `json:"is_photon"`
	DeviceID          string     `json:"device_id"`
	BinWidth          float64    `json:"bin_width"`
	NDataPoints       int        `json:"n_data_points"`
	Signal            []float64  `json:"signal"`
}
