package dto

type GlueExperimentBody struct {
	Wavelengths  []float64 `json:"wavelengths" binding:"required,min=1"`
	Polarization string    `json:"polarization"`
	H1           float64   `json:"h1" binding:"required"`
	H2           float64   `json:"h2" binding:"required"`
}
