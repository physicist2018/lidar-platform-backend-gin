package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/kshmirko/lidar-platform-go/internal/utils/response"
)

type PreparedExperimentRepositoryImpl struct {
	DataSource persistance.PreparedExperimentDataSource
	Log        *logrus.Logger
}

var _ domainRepo.PreparedExperimentRepository = (*PreparedExperimentRepositoryImpl)(nil)

func NewPreparedExperimentRepositoryImpl(ds persistance.PreparedExperimentDataSource, log *logrus.Logger) *PreparedExperimentRepositoryImpl {
	return &PreparedExperimentRepositoryImpl{DataSource: ds, Log: log}
}

func (r *PreparedExperimentRepositoryImpl) Create(ctx context.Context, exp *entity.PreparedExperiment) error {
	op := "PreparedExperimentRepository.Create"
	if err := r.DataSource.Create(ctx, exp); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *PreparedExperimentRepositoryImpl) Update(ctx context.Context, exp *entity.PreparedExperiment) error {
	op := "PreparedExperimentRepository.Update"
	if err := r.DataSource.Update(ctx, exp); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *PreparedExperimentRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.PreparedExperiment, error) {
	op := "PreparedExperimentRepository.FindByID"
	exp, err := r.DataSource.GetByID(ctx, id)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return exp, nil
}

func (r *PreparedExperimentRepositoryImpl) FindByExperimentID(ctx context.Context, experimentID uint) (*entity.PreparedExperiment, error) {
	op := "PreparedExperimentRepository.FindByExperimentID"
	exp, err := r.DataSource.GetByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return exp, nil
}
