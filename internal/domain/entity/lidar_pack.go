package entity

import "time"

// LidarPack represents a measurement session (one LicelPack archive).
type LidarPack struct {
	ID           uint        `json:"id"`
	ExperimentID uint        `json:"experiment_id"`
	StartTime    *time.Time  `json:"start_time,omitempty"`
	StopTime     *time.Time  `json:"stop_time,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	Files        []LidarFile `json:"files,omitempty"`
}

// LidarFile represents a single measurement file from the archive.
type LidarFile struct {
	ID           uint           `json:"id"`
	PackID       uint           `json:"pack_id"`
	Filename     string         `json:"filename"`
	Site         string         `json:"site,omitempty"`
	StartTime    *time.Time     `json:"start_time,omitempty"`
	StopTime     *time.Time     `json:"stop_time,omitempty"`
	Altitude     float64        `json:"altitude,omitempty"`
	Longitude    float64        `json:"longitude,omitempty"`
	Latitude     float64        `json:"latitude,omitempty"`
	Zenith       float64        `json:"zenith,omitempty"`
	Laser1NShots int            `json:"laser1_nshots,omitempty"`
	Laser1Freq   int            `json:"laser1_freq,omitempty"`
	Laser2NShots int            `json:"laser2_nshots,omitempty"`
	Laser2Freq   int            `json:"laser2_freq,omitempty"`
	Laser3NShots int            `json:"laser3_nshots,omitempty"`
	Laser3Freq   int            `json:"laser3_freq,omitempty"`
	NDatasets    int            `json:"ndatasets,omitempty"`
	Profiles     []LidarProfile `json:"profiles,omitempty"`
}

// LidarProfile represents a single measurement channel (profile).
type LidarProfile struct {
	ID           uint      `json:"id"`
	FileID       uint      `json:"file_id"`
	Active       bool      `json:"active"`
	IsPhoton     bool      `json:"is_photon"`
	LaserType    int       `json:"laser_type,omitempty"`
	NDataPoints  int       `json:"n_data_points,omitempty"`
	Reserved     []int     `json:"reserved,omitempty"`
	HighVoltage  int       `json:"high_voltage,omitempty"`
	BinWidth     float64   `json:"bin_width,omitempty"`
	Wavelength   float64   `json:"wavelength,omitempty"`
	Polarization string    `json:"polarization,omitempty"`
	BinShift     int       `json:"bin_shift,omitempty"`
	DecBinShift  int       `json:"dec_bin_shift,omitempty"`
	AdcBits      int       `json:"adc_bits,omitempty"`
	NShots       int       `json:"n_shots,omitempty"`
	DiscrLevel   float64   `json:"discr_level,omitempty"`
	DeviceID     string    `json:"device_id,omitempty"`
	NCrate       int       `json:"n_crate,omitempty"`
	Signal       []float64 `json:"signal,omitempty"`
}
