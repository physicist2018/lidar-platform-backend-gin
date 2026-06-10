package dto

type GlueExperimentBody struct {
	Wavelengths  []float64 `json:"wavelengths" validate:"required,min=1"`
	Polarization string    `json:"polarization"`
	H1           float64   `json:"h1" validate:"required"`
	H2           float64   `json:"h2" validate:"required"`
}
