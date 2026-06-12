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
	FindByExperimentIDAndAlgorithm(ctx context.Context, experimentID uint, algorithm string) ([]entity.ProcessingRun, error)
	// DeleteCascade deletes a set of processing runs and all their dependents recursively,
	// including their processed_signals.
	DeleteCascade(ctx context.Context, rootIDs []uint) error
}
