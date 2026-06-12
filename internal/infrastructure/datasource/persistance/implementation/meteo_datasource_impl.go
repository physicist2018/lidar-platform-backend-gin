package implementation

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/meteo"
)

var _ persistance.MeteoDataSource = (*MeteoDataSourceImpl)(nil)

type MeteoDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewMeteoDataSourceImpl(db *gorm.DB, log *logrus.Logger) *MeteoDataSourceImpl {
	return &MeteoDataSourceImpl{DB: db, Log: log}
}

func (d *MeteoDataSourceImpl) Save(ctx context.Context, record *entity.MeteoRecord) error {
	dbRecord := toMeteoDBEntity(record)
	if err := d.DB.WithContext(ctx).Create(dbRecord).Error; err != nil {
		d.Log.WithError(err).Error("MeteoDataSource.Save failed")
		return fmt.Errorf("save meteo record: %w", err)
	}
	record.ID = dbRecord.ID
	d.Log.WithFields(logrus.Fields{
		"meteo_id":      record.ID,
		"experiment_id": record.ExperimentID,
		"levels":        len(record.Pres),
	}).Info("meteo record saved")
	return nil
}

func (d *MeteoDataSourceImpl) FindByExperimentID(ctx context.Context, experimentID uint) (*entity.MeteoRecord, error) {
	var dbRecord dbEntity.MeteoRecordEntity
	if err := d.DB.WithContext(ctx).Where("experiment_id = ?", experimentID).First(&dbRecord).Error; err != nil {
		return nil, err
	}
	record := toMeteoDomain(&dbRecord)
	return &record, nil
}

func toMeteoDBEntity(record *entity.MeteoRecord) *dbEntity.MeteoRecordEntity {
	e := &dbEntity.MeteoRecordEntity{
		ExperimentID: record.ExperimentID,
		Pres:         meteo.Float64Slice(record.Pres),
		Hght:         meteo.Float64Slice(record.Hght),
		Temp:         meteo.Float64Slice(record.Temp),
	}
	if record.Relh != nil {
		e.Relh = meteo.Float64Slice(record.Relh)
	}
	if record.Mixr != nil {
		e.Mixr = meteo.Float64Slice(record.Mixr)
	}
	if record.Drct != nil {
		e.Drct = meteo.Float64Slice(record.Drct)
	}
	if record.Sknt != nil {
		e.Sknt = meteo.Float64Slice(record.Sknt)
	}
	return e
}

func toMeteoDomain(e *dbEntity.MeteoRecordEntity) entity.MeteoRecord {
	r := entity.MeteoRecord{
		ID:           e.ID,
		ExperimentID: e.ExperimentID,
		Pres:         []float64(e.Pres),
		Hght:         []float64(e.Hght),
		Temp:         []float64(e.Temp),
	}
	if e.Relh != nil {
		r.Relh = []float64(e.Relh)
	}
	if e.Mixr != nil {
		r.Mixr = []float64(e.Mixr)
	}
	if e.Drct != nil {
		r.Drct = []float64(e.Drct)
	}
	if e.Sknt != nil {
		r.Sknt = []float64(e.Sknt)
	}
	return r
}
