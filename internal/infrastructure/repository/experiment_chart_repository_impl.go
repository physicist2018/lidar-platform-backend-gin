package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/response"
)

var _ domainRepo.ExperimentChartRepository = (*ExperimentChartRepositoryImpl)(nil)

type ExperimentChartRepositoryImpl struct {
	DataSource persistance.ExperimentChartDataSource
	Log        *logrus.Logger
}

func NewExperimentChartRepositoryImpl(ds persistance.ExperimentChartDataSource, log *logrus.Logger) *ExperimentChartRepositoryImpl {
	return &ExperimentChartRepositoryImpl{DataSource: ds, Log: log}
}

func (r *ExperimentChartRepositoryImpl) Create(ctx context.Context, chart *entity.ExperimentChart) error {
	op := "ExperimentChartRepository.Create"
	if err := r.DataSource.Create(ctx, chart); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *ExperimentChartRepositoryImpl) FindByParams(
	ctx context.Context,
	experimentID uint,
	chartType, formula string,
	wavelen float64,
	polarization string,
	isPhoton int8,
	glued int8,
) (*entity.ExperimentChart, error) {
	op := "ExperimentChartRepository.FindByParams"
	chart, err := r.DataSource.FindByParams(ctx, experimentID, chartType, formula, wavelen, polarization, isPhoton, glued)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return chart, nil
}
