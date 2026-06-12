package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/response"
)

type ProcessedSignalRepositoryImpl struct {
	DataSource persistance.ProcessedSignalDataSource
	Log        *logrus.Logger
}

var _ domainRepo.ProcessedSignalRepository = (*ProcessedSignalRepositoryImpl)(nil)

func NewProcessedSignalRepositoryImpl(ds persistance.ProcessedSignalDataSource, log *logrus.Logger) *ProcessedSignalRepositoryImpl {
	return &ProcessedSignalRepositoryImpl{DataSource: ds, Log: log}
}

func (r *ProcessedSignalRepositoryImpl) BatchCreate(ctx context.Context, signals []entity.ProcessedSignal) error {
	op := "ProcessedSignalRepository.BatchCreate"
	if err := r.DataSource.BatchCreate(ctx, signals); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}

func (r *ProcessedSignalRepositoryImpl) FindByProcessingRunID(ctx context.Context, runID uint) ([]entity.ProcessedSignal, error) {
	op := "ProcessedSignalRepository.FindByProcessingRunID"
	signals, err := r.DataSource.GetByProcessingRunID(ctx, runID)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return signals, nil
}
