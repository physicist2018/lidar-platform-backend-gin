package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type ExperimentChartDataSource interface {
	Create(ctx context.Context, chart *entity.ExperimentChart) error
	FindByParams(ctx context.Context, experimentID uint, chartType, formula string, wavelen float64, polarization string, isPhoton int8, glued int8) (*entity.ExperimentChart, error)
}
