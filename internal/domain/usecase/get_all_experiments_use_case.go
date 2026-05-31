package usecase

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
)

type GetAllExperimentsUseCase interface {
	Execute(ctx context.Context, filter *entity.ExperimentFilter) (*pagination.Pagination[entity.Experiment], error)
}
