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
		isPhoton int8,
		polarization string,
		vizType string,
		outputType string,
		formula string,
		regenerate bool,
		glued int8,
	) (string, error) // returns presigned Minio URL
}
