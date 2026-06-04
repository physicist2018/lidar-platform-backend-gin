package usecase

import "context"

type GluePreparedExperimentUseCase interface {
	Execute(ctx context.Context, experimentID uint, wavelengths []float64, polarization string, h1, h2 float64) error
}
