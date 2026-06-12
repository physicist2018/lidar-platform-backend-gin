package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/response"
)

type MeteoRepositoryImpl struct {
	DataSource persistance.MeteoDataSource
	Log        *logrus.Logger
}

var _ domainRepo.MeteoRepository = (*MeteoRepositoryImpl)(nil)

func NewMeteoRepositoryImpl(ds persistance.MeteoDataSource, log *logrus.Logger) *MeteoRepositoryImpl {
	return &MeteoRepositoryImpl{DataSource: ds, Log: log}
}

func (r *MeteoRepositoryImpl) Save(ctx context.Context, record *entity.MeteoRecord) error {
	op := "MeteoRepository.Save"
	if err := r.DataSource.Save(ctx, record); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *MeteoRepositoryImpl) FindByExperimentID(ctx context.Context, experimentID uint) (*entity.MeteoRecord, error) {
	op := "MeteoRepository.FindByExperimentID"
	record, err := r.DataSource.FindByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return record, nil
}
