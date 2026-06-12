package usecase

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type GetExperimentByIDUseCase interface {
	Execute(ctx context.Context, id uint) (*entity.Experiment, error)
}
