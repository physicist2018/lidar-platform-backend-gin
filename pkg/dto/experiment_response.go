package dto

import "time"

type ExperimentResponse struct {
	ID                   uint       `json:"id"`
	Title                string     `json:"title"`
	Comments             string     `json:"comments"`
	MeasurementStartTime *time.Time `json:"measurement_start_time,omitempty"`
	MeasurementStopTime  *time.Time `json:"measurement_stop_time,omitempty"`
	LicelZipPath         string     `json:"licel_zip_path"`
	LicelBgrPath         string     `json:"licel_bgr_path"`
	MeteoFilePath        string     `json:"meteo_file_path"`
	Status               string     `json:"status"`
	ErrorMsg             string     `json:"error_msg,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}
