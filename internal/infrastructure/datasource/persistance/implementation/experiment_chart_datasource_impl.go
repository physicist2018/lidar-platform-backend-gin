package implementation

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance"
)

var _ persistance.ExperimentChartDataSource = (*ExperimentChartDataSourceImpl)(nil)

type ExperimentChartDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewExperimentChartDataSourceImpl(db *gorm.DB, log *logrus.Logger) *ExperimentChartDataSourceImpl {
	return &ExperimentChartDataSourceImpl{DB: db, Log: log}
}

func (d *ExperimentChartDataSourceImpl) Create(ctx context.Context, chart *entity.ExperimentChart) error {
	dbChart := &dbEntity.ExperimentChartEntity{
		ExperimentID: chart.ExperimentID,
		ChartType:    chart.ChartType,
		Formula:      chart.Formula,
		Wavelen:      chart.Wavelen,
		Polarization: chart.Polarization,
		IsPhoton:     chart.IsPhoton,
		PathToObject: chart.PathToObject,
	}
	if err := d.DB.WithContext(ctx).Create(dbChart).Error; err != nil {
		d.Log.WithError(err).Error("ExperimentChartDataSource.Create failed")
		return err
	}
	chart.ID = dbChart.ID
	return nil
}

func (d *ExperimentChartDataSourceImpl) FindByParams(
	ctx context.Context,
	experimentID uint,
	chartType, formula string,
	wavelen float64,
	polarization string,
	isPhoton int8,
) (*entity.ExperimentChart, error) {
	var dbChart dbEntity.ExperimentChartEntity
	err := d.DB.WithContext(ctx).
		Where("experiment_id = ?", experimentID).
		Where("chart_type = ?", chartType).
		Where("formula = ?", formula).
		Where("wavelen = ?", wavelen).
		Where("polarization = ?", polarization).
		Where("is_photon = ?", isPhoton).
		First(&dbChart).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		d.Log.WithError(err).Error("ExperimentChartDataSource.FindByParams failed")
		return nil, err
	}
	return toChartDomain(&dbChart), nil
}

func toChartDomain(dbChart *dbEntity.ExperimentChartEntity) *entity.ExperimentChart {
	return &entity.ExperimentChart{
		ID:           dbChart.ID,
		ExperimentID: dbChart.ExperimentID,
		ChartType:    dbChart.ChartType,
		Formula:      dbChart.Formula,
		Wavelen:      dbChart.Wavelen,
		Polarization: dbChart.Polarization,
		IsPhoton:     dbChart.IsPhoton,
		PathToObject: dbChart.PathToObject,
	}
}
