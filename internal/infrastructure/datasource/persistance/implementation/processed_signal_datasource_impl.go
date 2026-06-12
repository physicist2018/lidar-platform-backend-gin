package implementation

import (
	"context"

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
			ProcessingRunID:   s.ProcessingRunID,
			OriginalProfileID: s.OriginalProfileID,
			Wavelength:        s.Wavelength,
			Polarization:      s.Polarization,
			IsPhoton:          s.IsPhoton,
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
			Wavelength:        e.Wavelength,
			Polarization:      e.Polarization,
			IsPhoton:          e.IsPhoton,
			Signal:            []float64(e.Signal),
		}
	}
	return signals, nil
}
