package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type ProcessingRunRepository interface {
	Create(ctx context.Context, run *entity.ProcessingRun) error
	Update(ctx context.Context, run *entity.ProcessingRun) error
	FindByID(ctx context.Context, id uint) (*entity.ProcessingRun, error)
	FindByExperimentID(ctx context.Context, experimentID uint) ([]entity.ProcessingRun, error)
}
