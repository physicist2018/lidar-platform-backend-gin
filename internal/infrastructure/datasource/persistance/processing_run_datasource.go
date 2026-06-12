package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type ProcessingRunDataSource interface {
	Create(ctx context.Context, run *entity.ProcessingRun) error
	Update(ctx context.Context, run *entity.ProcessingRun) error
	GetByID(ctx context.Context, id uint) (*entity.ProcessingRun, error)
	GetByExperimentID(ctx context.Context, experimentID uint) ([]entity.ProcessingRun, error)
	GetByExperimentIDAndAlgorithm(ctx context.Context, experimentID uint, algorithm string) ([]entity.ProcessingRun, error)
	GetByDependsOnID(ctx context.Context, parentID uint) ([]entity.ProcessingRun, error)
	DeleteByIDs(ctx context.Context, ids []uint) error
}
