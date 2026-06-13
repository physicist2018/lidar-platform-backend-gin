package entity

import "time"

// ProcessedSignalFilter defines optional filters for querying processed signals.
type ProcessedSignalFilter struct {
	Wavelength   *float64
	Polarization *string
	DeviceID     *string
	TimeFrom     *time.Time
	TimeTo       *time.Time
}
