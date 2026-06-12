package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type PreparedExperimentRepository interface {
	Create(ctx context.Context, exp *entity.PreparedExperiment) error
	Update(ctx context.Context, exp *entity.PreparedExperiment) error
	FindByID(ctx context.Context, id uint) (*entity.PreparedExperiment, error)
	FindByExperimentID(ctx context.Context, experimentID uint) (*entity.PreparedExperiment, error)
}
