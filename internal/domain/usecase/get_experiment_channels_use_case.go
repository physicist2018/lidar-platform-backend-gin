package usecase

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type GetExperimentChannelsUseCase interface {
	Execute(ctx context.Context, id uint) ([]entity.ExperimentChannel, error)
}
