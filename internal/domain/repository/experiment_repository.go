package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type ExperimentRepository interface {
	Create(ctx context.Context, exp *entity.Experiment) error
	Update(ctx context.Context, exp *entity.Experiment) error
	FindByID(ctx context.Context, id uint) (*entity.Experiment, error)
	FindAll(ctx context.Context, filter *entity.ExperimentFilter) (*pagination.Pagination[entity.Experiment], error)
}
