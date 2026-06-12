package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/response"
)

type ProcessingRunRepositoryImpl struct {
	DataSource persistance.ProcessingRunDataSource
	Log        *logrus.Logger
}

var _ domainRepo.ProcessingRunRepository = (*ProcessingRunRepositoryImpl)(nil)

func NewProcessingRunRepositoryImpl(ds persistance.ProcessingRunDataSource, log *logrus.Logger) *ProcessingRunRepositoryImpl {
	return &ProcessingRunRepositoryImpl{DataSource: ds, Log: log}
}

func (r *ProcessingRunRepositoryImpl) Create(ctx context.Context, run *entity.ProcessingRun) error {
	op := "ProcessingRunRepository.Create"
	if err := r.DataSource.Create(ctx, run); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *ProcessingRunRepositoryImpl) Update(ctx context.Context, run *entity.ProcessingRun) error {
	op := "ProcessingRunRepository.Update"
	if err := r.DataSource.Update(ctx, run); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *ProcessingRunRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.ProcessingRun, error) {
	op := "ProcessingRunRepository.FindByID"
	run, err := r.DataSource.GetByID(ctx, id)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return run, nil
}

func (r *ProcessingRunRepositoryImpl) FindByExperimentID(ctx context.Context, experimentID uint) ([]entity.ProcessingRun, error) {
	op := "ProcessingRunRepository.FindByExperimentID"
	runs, err := r.DataSource.GetByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return runs, nil
}
