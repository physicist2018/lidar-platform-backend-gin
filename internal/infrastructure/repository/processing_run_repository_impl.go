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
	DataSource       persistance.ProcessingRunDataSource
	SignalDataSource persistance.ProcessedSignalDataSource
	Log              *logrus.Logger
}

var _ domainRepo.ProcessingRunRepository = (*ProcessingRunRepositoryImpl)(nil)

func NewProcessingRunRepositoryImpl(
	ds persistance.ProcessingRunDataSource,
	signalDS persistance.ProcessedSignalDataSource,
	log *logrus.Logger,
) *ProcessingRunRepositoryImpl {
	return &ProcessingRunRepositoryImpl{
		DataSource:       ds,
		SignalDataSource: signalDS,
		Log:              log,
	}
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

func (r *ProcessingRunRepositoryImpl) FindByExperimentIDAndAlgorithm(ctx context.Context, experimentID uint, algorithm string) ([]entity.ProcessingRun, error) {
	op := "ProcessingRunRepository.FindByExperimentIDAndAlgorithm"
	runs, err := r.DataSource.GetByExperimentIDAndAlgorithm(ctx, experimentID, algorithm)
	if err != nil {
		return nil, response.InternalError(op, err)
	}
	return runs, nil
}

func (r *ProcessingRunRepositoryImpl) DeleteCascade(ctx context.Context, rootIDs []uint) error {
	op := "ProcessingRunRepository.DeleteCascade"

	// Collect all IDs recursively: rootIDs → their dependents → their dependents → ...
	allIDs := make(map[uint]bool)
	queue := make([]uint, len(rootIDs))
	copy(queue, rootIDs)

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if allIDs[id] {
			continue
		}
		allIDs[id] = true

		children, err := r.DataSource.GetByDependsOnID(ctx, id)
		if err != nil {
			return response.InternalError(op, err)
		}
		for _, child := range children {
			if !allIDs[child.ID] {
				queue = append(queue, child.ID)
			}
		}
	}

	if len(allIDs) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(allIDs))
	for id := range allIDs {
		ids = append(ids, id)
	}

	// Remove processed signals first
	if err := r.SignalDataSource.DeleteByProcessingRunIDs(ctx, ids); err != nil {
		return response.InternalError(op, err)
	}

	// Then soft-delete the processing runs
	if err := r.DataSource.DeleteByIDs(ctx, ids); err != nil {
		return response.InternalError(op, err)
	}

	return nil
}
