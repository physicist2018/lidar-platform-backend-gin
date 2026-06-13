package usecase

import (
	"context"
	"time"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type GetStage0DataUseCase interface {
	Execute(
		ctx context.Context,
		runID uint,
		wavelength float64,
		polarization string,
		deviceID string,
		timeFrom *time.Time,
		timeTo *time.Time,
	) (*entity.Stage0DataResult, error)
}
