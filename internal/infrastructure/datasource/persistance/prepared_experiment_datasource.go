package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type PreparedExperimentDataSource interface {
	Create(ctx context.Context, exp *entity.PreparedExperiment) error
	Update(ctx context.Context, exp *entity.PreparedExperiment) error
	GetByID(ctx context.Context, id uint) (*entity.PreparedExperiment, error)
	GetByExperimentID(ctx context.Context, experimentID uint) (*entity.PreparedExperiment, error)
}
