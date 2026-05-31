package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
	"github.com/kshmirko/lidar-platform-go/internal/utils/response"
)

type ExperimentRepositoryImpl struct {
	DataSource persistance.ExperimentDataSource
	Log        *logrus.Logger
}

var _ domainRepo.ExperimentRepository = (*ExperimentRepositoryImpl)(nil)

func NewExperimentRepositoryImpl(ds persistance.ExperimentDataSource, log *logrus.Logger) *ExperimentRepositoryImpl {
	return &ExperimentRepositoryImpl{DataSource: ds, Log: log}
}

func (r *ExperimentRepositoryImpl) Create(ctx context.Context, exp *entity.Experiment) error {
	op := "ExperimentRepository.Create"
	if err := r.DataSource.Create(ctx, exp); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *ExperimentRepositoryImpl) Update(ctx context.Context, exp *entity.Experiment) error {
	op := "ExperimentRepository.Update"
	if err := r.DataSource.Update(ctx, exp); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *ExperimentRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.Experiment, error) {
	op := "ExperimentRepository.FindByID"
	exp, err := r.DataSource.GetByID(ctx, id)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return exp, nil
}

func (r *ExperimentRepositoryImpl) FindAll(ctx context.Context, filter *entity.ExperimentFilter) (*pagination.Pagination[entity.Experiment], error) {
	op := "ExperimentRepository.FindAll"
	exps, total, err := r.DataSource.GetAll(ctx, filter)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return pagination.New(exps, total, filter.Page, filter.Limit), nil
}
