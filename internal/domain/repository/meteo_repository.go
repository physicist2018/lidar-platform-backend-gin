package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type MeteoRepository interface {
	Save(ctx context.Context, record *entity.MeteoRecord) error
	FindByExperimentID(ctx context.Context, experimentID uint) (*entity.MeteoRecord, error)
}
