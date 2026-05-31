package persistance

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type ExperimentDataSource interface {
	Create(ctx context.Context, exp *entity.Experiment) error
	Update(ctx context.Context, exp *entity.Experiment) error
	GetByID(ctx context.Context, id uint) (*entity.Experiment, error)
	GetAll(ctx context.Context, filter *entity.ExperimentFilter) ([]entity.Experiment, int64, error)
}
