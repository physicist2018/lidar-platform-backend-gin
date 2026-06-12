package implementation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

var _ persistance.ExperimentDataSource = (*ExperimentDataSourceImpl)(nil)

type ExperimentDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewExperimentDataSourceImpl(db *gorm.DB, log *logrus.Logger) *ExperimentDataSourceImpl {
	return &ExperimentDataSourceImpl{DB: db, Log: log}
}

func (d *ExperimentDataSourceImpl) Create(ctx context.Context, exp *entity.Experiment) error {
	dbExp := &dbEntity.ExperimentEntity{
		UserID:   exp.UserID,
		Title:    exp.Title,
		Comments: exp.Comments,
		Status:   string(exp.Status),
	}
	if err := d.DB.WithContext(ctx).Create(dbExp).Error; err != nil {
		d.Log.WithError(err).Error("ExperimentDataSource.Create failed")
		return err
	}
	exp.ID = dbExp.ID
	exp.CreatedAt = dbExp.CreatedAt
	exp.UpdatedAt = dbExp.UpdatedAt
	return nil
}

func (d *ExperimentDataSourceImpl) Update(ctx context.Context, exp *entity.Experiment) error {
	updates := map[string]interface{}{
		"title":  exp.Title,
		"status": string(exp.Status),
	}

	if exp.UserID != 0 {
		updates["user_id"] = exp.UserID
	}

	if exp.Comments != "" {
		updates["comments"] = exp.Comments
	}
	if exp.MeasurementStartTime != nil {
		updates["measurement_start_time"] = *exp.MeasurementStartTime
	}
	if exp.MeasurementStopTime != nil {
		updates["measurement_stop_time"] = *exp.MeasurementStopTime
	}
	if exp.LidarPackID != nil {
		updates["lidar_pack_id"] = *exp.LidarPackID
	}
	if exp.BgrFileID != nil {
		updates["bgr_file_id"] = *exp.BgrFileID
	}
	if exp.MeteoID != nil {
		updates["meteo_id"] = *exp.MeteoID
	}
	if exp.LicelZipPath != "" {
		updates["licel_zip_path"] = exp.LicelZipPath
	}
	if exp.LicelBgrPath != "" {
		updates["licel_bgr_path"] = exp.LicelBgrPath
	}
	if exp.MeteoFilePath != "" {
		updates["meteo_file_path"] = exp.MeteoFilePath
	}
	if exp.ErrorMsg != "" {
		updates["error_msg"] = exp.ErrorMsg
	}
	if len(exp.AvailableChannels) > 0 {
		channelsJSON, err := json.Marshal(exp.AvailableChannels)
		if err != nil {
			d.Log.WithError(err).Error("ExperimentDataSource.Update: marshal available_channels")
		} else {
			updates["available_channels"] = channelsJSON
		}
	}

	result := d.DB.WithContext(ctx).Model(&dbEntity.ExperimentEntity{}).Where("id = ?", exp.ID).Updates(updates)
	if result.Error != nil {
		d.Log.WithError(result.Error).Error("ExperimentDataSource.Update failed")
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("experiment with id %d not found", exp.ID)
	}
	return nil
}

func (d *ExperimentDataSourceImpl) GetByID(ctx context.Context, id uint) (*entity.Experiment, error) {
	var dbExp dbEntity.ExperimentEntity
	if err := d.DB.WithContext(ctx).First(&dbExp, id).Error; err != nil {
		return nil, err
	}

	exp := toExperimentDomain(&dbExp)
	return &exp, nil
}

func (d *ExperimentDataSourceImpl) GetAll(ctx context.Context, filter *entity.ExperimentFilter) ([]entity.Experiment, int64, error) {
	var dbExps []dbEntity.ExperimentEntity
	var total int64

	query := d.DB.WithContext(ctx).Model(&dbEntity.ExperimentEntity{})

	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	if filter.Title != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Title+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		d.Log.WithError(err).Error("ExperimentDataSource.GetAll: count failed")
		return nil, 0, err
	}

	order := "id DESC"
	if filter.Sort == "asc" {
		order = "id ASC"
	}

	offset := pagination.Offset(filter.Page, filter.Limit)

	if err := query.Order(order).Offset(offset).Limit(filter.Limit).Find(&dbExps).Error; err != nil {
		d.Log.WithError(err).Error("ExperimentDataSource.GetAll: find failed")
		return nil, 0, err
	}

	exps := make([]entity.Experiment, len(dbExps))
	for i, e := range dbExps {
		exps[i] = toExperimentDomain(&e)
	}

	return exps, total, nil
}

func toExperimentDomain(dbExp *dbEntity.ExperimentEntity) entity.Experiment {
	exp := entity.Experiment{
		ID:                   dbExp.ID,
		UserID:               dbExp.UserID,
		Title:                dbExp.Title,
		Comments:             dbExp.Comments,
		MeasurementStartTime: dbExp.MeasurementStartTime,
		MeasurementStopTime:  dbExp.MeasurementStopTime,
		LidarPackID:          dbExp.LidarPackID,
		BgrFileID:            dbExp.BgrFileID,
		MeteoID:              dbExp.MeteoID,
		LicelZipPath:         dbExp.LicelZipPath,
		LicelBgrPath:         dbExp.LicelBgrPath,
		MeteoFilePath:        dbExp.MeteoFilePath,
		Status:               entity.ExperimentStatus(dbExp.Status),
		ErrorMsg:             dbExp.ErrorMsg,
		CreatedAt:            dbExp.CreatedAt,
		UpdatedAt:            dbExp.UpdatedAt,
	}
	_ = json.Unmarshal(dbExp.AvailableChannels, &exp.AvailableChannels)
	return exp
}
