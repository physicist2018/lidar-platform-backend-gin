package dto

// ExperimentChannelResponse is the DTO for a single measurement channel.
type ExperimentChannelResponse struct {
	Wavelength   float64 `json:"wavelen"`
	Polarization string  `json:"polarization"`
	IsPhoton     int     `json:"isPhoton"` // 0 = analog, 1 = photon
	IsActive     int     `json:"isActive"` // 0 = no signal, 1 = has signal
}

// ExperimentChannelsResponse wraps the list of channels.
type ExperimentChannelsResponse struct {
	Channels []ExperimentChannelResponse `json:"channels"`
}
