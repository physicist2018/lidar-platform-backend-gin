package repository

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type ExperimentChartRepository interface {
	Create(ctx context.Context, chart *entity.ExperimentChart) error
	FindByParams(ctx context.Context, experimentID uint, chartType, formula string, wavelen float64, polarization string, isPhoton int8) (*entity.ExperimentChart, error)
}
