package usecase

import "context"

// VisualizeResult holds the generated visualization data.
type VisualizeResult struct {
	ContentType string
	Body        []byte
}

type VisualizePreparedExperimentUseCase interface {
	Execute(
		ctx context.Context,
		prepID uint,
		wavelen float64,
		isPhoton bool,
		polarization string,
		vizType string, // "image" or "profile"
		outputType string, // "svg" or "json"
	) (*VisualizeResult, error)
}
