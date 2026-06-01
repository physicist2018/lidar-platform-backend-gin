package repository

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type PreparedExperimentRepository interface {
	Create(ctx context.Context, exp *entity.PreparedExperiment) error
	Update(ctx context.Context, exp *entity.PreparedExperiment) error
	FindByID(ctx context.Context, id uint) (*entity.PreparedExperiment, error)
	FindByExperimentID(ctx context.Context, experimentID uint) (*entity.PreparedExperiment, error)
}
