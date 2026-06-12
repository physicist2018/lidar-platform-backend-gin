package usecase

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type GetAllExperimentsUseCase interface {
	Execute(ctx context.Context, filter *entity.ExperimentFilter) (*pagination.Pagination[entity.Experiment], error)
}
