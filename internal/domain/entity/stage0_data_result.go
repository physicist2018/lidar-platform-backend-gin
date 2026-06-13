package entity

import "time"

// Stage0DataResult holds the processed result data for the stage0 data endpoint.
type Stage0DataResult struct {
	Distance []float64
	Data     [][]float64
	Time     []time.Time
}
