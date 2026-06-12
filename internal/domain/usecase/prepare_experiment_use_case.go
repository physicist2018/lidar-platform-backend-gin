package usecase

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type PrepareExperimentUseCase interface {
	Execute(ctx context.Context, userID, experimentID uint, cropAlt float64, bgrType entity.BGRType, bgrAlt float64) (*entity.PreparedExperiment, error)
}
