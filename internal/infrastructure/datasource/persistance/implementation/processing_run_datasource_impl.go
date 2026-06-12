package implementation

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
)

var _ persistance.ProcessingRunDataSource = (*ProcessingRunDataSourceImpl)(nil)

type ProcessingRunDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewProcessingRunDataSourceImpl(db *gorm.DB, log *logrus.Logger) *ProcessingRunDataSourceImpl {
	return &ProcessingRunDataSourceImpl{DB: db, Log: log}
}

func (d *ProcessingRunDataSourceImpl) Create(ctx context.Context, run *entity.ProcessingRun) error {
	dbRun := &dbEntity.ProcessingRunEntity{
		ExperimentID: run.ExperimentID,
		UserID:       run.UserID,
		Algorithm:    run.Algorithm,
		Params:       run.Params,
		DependsOnID:  run.DependsOnID,
		Status:       string(run.Status),
	}
	if err := d.DB.WithContext(ctx).Create(dbRun).Error; err != nil {
		d.Log.WithError(err).Error("ProcessingRunDataSource.Create failed")
		return err
	}
	run.ID = dbRun.ID
	run.CreatedAt = dbRun.CreatedAt
	run.UpdatedAt = dbRun.UpdatedAt
	return nil
}

func (d *ProcessingRunDataSourceImpl) Update(ctx context.Context, run *entity.ProcessingRun) error {
	updates := map[string]interface{}{}

	if run.Status != "" {
		updates["status"] = string(run.Status)
	}
	if run.ErrorMsg != "" {
		updates["error_msg"] = run.ErrorMsg
	}
	if run.Params != nil {
		updates["params"] = run.Params
	}

	result := d.DB.WithContext(ctx).
		Model(&dbEntity.ProcessingRunEntity{}).
		Where("id = ?", run.ID).
		Updates(updates)
	if result.Error != nil {
		d.Log.WithError(result.Error).Error("ProcessingRunDataSource.Update failed")
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("processing run with id %d not found", run.ID)
	}
	return nil
}

func (d *ProcessingRunDataSourceImpl) GetByID(ctx context.Context, id uint) (*entity.ProcessingRun, error) {
	var dbRun dbEntity.ProcessingRunEntity
	if err := d.DB.WithContext(ctx).First(&dbRun, id).Error; err != nil {
		return nil, err
	}
	run := toProcessingRunDomain(&dbRun)
	return &run, nil
}

func (d *ProcessingRunDataSourceImpl) GetByExperimentID(ctx context.Context, experimentID uint) ([]entity.ProcessingRun, error) {
	var dbRuns []dbEntity.ProcessingRunEntity
	if err := d.DB.WithContext(ctx).
		Where("experiment_id = ?", experimentID).
		Order("id DESC").
		Find(&dbRuns).Error; err != nil {
		return nil, err
	}
	runs := make([]entity.ProcessingRun, len(dbRuns))
	for i := range dbRuns {
		runs[i] = toProcessingRunDomain(&dbRuns[i])
	}
	return runs, nil
}

func (d *ProcessingRunDataSourceImpl) GetByExperimentIDAndAlgorithm(ctx context.Context, experimentID uint, algorithm string) ([]entity.ProcessingRun, error) {
	var dbRuns []dbEntity.ProcessingRunEntity
	if err := d.DB.WithContext(ctx).
		Where("experiment_id = ? AND algorithm = ?", experimentID, algorithm).
		Order("id DESC").
		Find(&dbRuns).Error; err != nil {
		return nil, err
	}
	runs := make([]entity.ProcessingRun, len(dbRuns))
	for i := range dbRuns {
		runs[i] = toProcessingRunDomain(&dbRuns[i])
	}
	return runs, nil
}

func (d *ProcessingRunDataSourceImpl) GetByDependsOnID(ctx context.Context, parentID uint) ([]entity.ProcessingRun, error) {
	var dbRuns []dbEntity.ProcessingRunEntity
	if err := d.DB.WithContext(ctx).
		Where("depends_on_id = ?", parentID).
		Find(&dbRuns).Error; err != nil {
		return nil, err
	}
	runs := make([]entity.ProcessingRun, len(dbRuns))
	for i := range dbRuns {
		runs[i] = toProcessingRunDomain(&dbRuns[i])
	}
	return runs, nil
}

func (d *ProcessingRunDataSourceImpl) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := d.DB.WithContext(ctx).
		Delete(&dbEntity.ProcessingRunEntity{}, "id IN ?", ids).Error; err != nil {
		d.Log.WithError(err).Error("ProcessingRunDataSource.DeleteByIDs failed")
		return err
	}
	return nil
}

func toProcessingRunDomain(dbRun *dbEntity.ProcessingRunEntity) entity.ProcessingRun {
	return entity.ProcessingRun{
		ID:           dbRun.ID,
		ExperimentID: dbRun.ExperimentID,
		UserID:       dbRun.UserID,
		Algorithm:    dbRun.Algorithm,
		Params:       dbRun.Params,
		DependsOnID:  dbRun.DependsOnID,
		Status:       entity.ProcessingStatus(dbRun.Status),
		ErrorMsg:     dbRun.ErrorMsg,
		CreatedAt:    dbRun.CreatedAt,
		UpdatedAt:    dbRun.UpdatedAt,
	}
}
