package implementation

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/gorm/datatypes"
)

var _ persistance.ProcessedSignalDataSource = (*ProcessedSignalDataSourceImpl)(nil)

type ProcessedSignalDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewProcessedSignalDataSourceImpl(db *gorm.DB, log *logrus.Logger) *ProcessedSignalDataSourceImpl {
	return &ProcessedSignalDataSourceImpl{DB: db, Log: log}
}

func (d *ProcessedSignalDataSourceImpl) BatchCreate(ctx context.Context, signals []entity.ProcessedSignal) error {
	entities := make([]dbEntity.ProcessedSignalEntity, len(signals))
	for i, s := range signals {
		entities[i] = dbEntity.ProcessedSignalEntity{
			ID:                s.ID,
			ProcessingRunID:   s.ProcessingRunID,
			OriginalProfileID: s.OriginalProfileID,
			FileID:            s.FileID,
			Wavelength:        s.Wavelength,
			Polarization:      s.Polarization,
			IsPhoton:          s.IsPhoton,
			DeviceID:          s.DeviceID,
			BinWidth:          s.BinWidth,
			NDataPoints:       s.NDataPoints,
			Signal:            datatypes.Float64Slice(s.Signal),
		}
	}

	if err := d.DB.WithContext(ctx).Create(&entities).Error; err != nil {
		d.Log.WithError(err).Error("ProcessedSignalDataSource.BatchCreate failed")
		return err
	}

	// Set IDs back
	for i := range signals {
		signals[i].ID = entities[i].ID
	}
	return nil
}

func (d *ProcessedSignalDataSourceImpl) GetByProcessingRunID(ctx context.Context, runID uint) ([]entity.ProcessedSignal, error) {
	var dbEntities []dbEntity.ProcessedSignalEntity
	if err := d.DB.WithContext(ctx).
		Where("processing_run_id = ?", runID).
		Find(&dbEntities).Error; err != nil {
		return nil, err
	}

	signals := make([]entity.ProcessedSignal, len(dbEntities))
	for i, e := range dbEntities {
		signals[i] = entity.ProcessedSignal{
			ID:                e.ID,
			ProcessingRunID:   e.ProcessingRunID,
			OriginalProfileID: e.OriginalProfileID,
			FileID:            e.FileID,
			Wavelength:        e.Wavelength,
			Polarization:      e.Polarization,
			IsPhoton:          e.IsPhoton,
			DeviceID:          e.DeviceID,
			BinWidth:          e.BinWidth,
			NDataPoints:       e.NDataPoints,
			Signal:            []float64(e.Signal),
		}
	}
	return signals, nil
}

func (d *ProcessedSignalDataSourceImpl) DeleteByProcessingRunIDs(ctx context.Context, runIDs []uint) error {
	if len(runIDs) == 0 {
		return nil
	}
	if err := d.DB.WithContext(ctx).
		Where("processing_run_id IN ?", runIDs).
		Delete(&dbEntity.ProcessedSignalEntity{}).Error; err != nil {
		d.Log.WithError(err).Error("ProcessedSignalDataSource.DeleteByProcessingRunIDs failed")
		return err
	}
	return nil
}

func (d *ProcessedSignalDataSourceImpl) GetByProcessingRunIDFiltered(
	ctx context.Context,
	runID uint,
	filter entity.ProcessedSignalFilter,
) ([]entity.ProcessedSignal, error) {
	var dbEntities []struct {
		dbEntity.ProcessedSignalEntity
		FileStartTime *time.Time `gorm:"column:file_start_time"`
	}

	query := d.DB.WithContext(ctx).
		Table("processed_signals").
		Select("processed_signals.*, lidar_files.start_time AS file_start_time").
		Joins("LEFT JOIN lidar_files ON lidar_files.id = processed_signals.file_id").
		Where("processed_signals.processing_run_id = ?", runID)

	if filter.Wavelength != nil {
		query = query.Where("processed_signals.wavelength = ?", *filter.Wavelength)
	}
	if filter.Polarization != nil {
		query = query.Where("processed_signals.polarization = ?", *filter.Polarization)
	}
	if filter.DeviceID != nil {
		query = query.Where("processed_signals.device_id = ?", *filter.DeviceID)
	}
	if filter.TimeFrom != nil {
		query = query.Where("(lidar_files.start_time >= ? OR processed_signals.file_id = 0)", *filter.TimeFrom)
	}
	if filter.TimeTo != nil {
		query = query.Where("(lidar_files.start_time <= ? OR processed_signals.file_id = 0)", *filter.TimeTo)
	}

	query = query.Order("lidar_files.start_time ASC NULLS LAST")

	if err := query.Find(&dbEntities).Error; err != nil {
		return nil, err
	}

	signals := make([]entity.ProcessedSignal, len(dbEntities))
	for i, e := range dbEntities {
		signals[i] = entity.ProcessedSignal{
			ID:                e.ID,
			ProcessingRunID:   e.ProcessingRunID,
			OriginalProfileID: e.OriginalProfileID,
			FileID:            e.FileID,
			FileStartTime:     e.FileStartTime,
			Wavelength:        e.Wavelength,
			Polarization:      e.Polarization,
			IsPhoton:          e.IsPhoton,
			DeviceID:          e.DeviceID,
			BinWidth:          e.BinWidth,
			NDataPoints:       e.NDataPoints,
			Signal:            []float64(e.Signal),
		}
	}
	return signals, nil
}
