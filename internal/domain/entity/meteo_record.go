package entity

// MeteoRecord holds all meteo levels for an experiment.
// Pres, Hght, Temp are always present (from file or standard atmosphere).
// Relh, Mixr, Drct, Sknt may be nil if data is unavailable.
type MeteoRecord struct {
	ID           uint
	ExperimentID uint
	Pres         []float64
	Hght         []float64
	Temp         []float64
	Relh         []float64
	Mixr         []float64
	Drct         []float64
	Sknt         []float64
}
