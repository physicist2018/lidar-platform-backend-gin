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

var _ persistance.PreparedExperimentDataSource = (*PreparedExperimentDataSourceImpl)(nil)

type PreparedExperimentDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewPreparedExperimentDataSourceImpl(db *gorm.DB, log *logrus.Logger) *PreparedExperimentDataSourceImpl {
	return &PreparedExperimentDataSourceImpl{DB: db, Log: log}
}

func (d *PreparedExperimentDataSourceImpl) Create(ctx context.Context, exp *entity.PreparedExperiment) error {
	dbExp := &dbEntity.PreparedExperimentEntity{
		UserID:       exp.UserID,
		ExperimentID: exp.ExperimentID,
		CropAlt:      exp.CropAlt,
		BGRType:      string(exp.BGRType),
		BGRAlt:       exp.BGRAlt,
		PathToData:   exp.PathToData,
		Status:       string(exp.Status),
	}
	if err := d.DB.WithContext(ctx).Create(dbExp).Error; err != nil {
		d.Log.WithError(err).Error("PreparedExperimentDataSource.Create failed")
		return err
	}
	exp.ID = dbExp.ID
	return nil
}

func (d *PreparedExperimentDataSourceImpl) Update(ctx context.Context, exp *entity.PreparedExperiment) error {
	updates := map[string]interface{}{}

	if exp.UserID != 0 {
		updates["user_id"] = exp.UserID
	}

	if exp.CropAlt != 0 {
		updates["crop_alt"] = exp.CropAlt
	}
	if exp.BGRType != "" {
		updates["bgr_type"] = string(exp.BGRType)
	}
	if exp.BGRAlt != 0 {
		updates["bgr_alt"] = exp.BGRAlt
	}
	if exp.PathToData != "" {
		updates["path_to_data"] = exp.PathToData
	}
	if exp.Status != "" {
		updates["status"] = string(exp.Status)
	}
	if exp.ErrorMsg != "" {
		updates["error_msg"] = exp.ErrorMsg
	}

	result := d.DB.WithContext(ctx).Model(&dbEntity.PreparedExperimentEntity{}).Where("id = ?", exp.ID).Updates(updates)
	if result.Error != nil {
		d.Log.WithError(result.Error).Error("PreparedExperimentDataSource.Update failed")
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("prepared experiment with id %d not found", exp.ID)
	}
	return nil
}

func (d *PreparedExperimentDataSourceImpl) GetByID(ctx context.Context, id uint) (*entity.PreparedExperiment, error) {
	var dbExp dbEntity.PreparedExperimentEntity
	if err := d.DB.WithContext(ctx).First(&dbExp, id).Error; err != nil {
		return nil, err
	}
	exp := toPreparedExperimentDomain(&dbExp)
	return &exp, nil
}

func (d *PreparedExperimentDataSourceImpl) GetByExperimentID(ctx context.Context, experimentID uint) (*entity.PreparedExperiment, error) {
	var dbExp dbEntity.PreparedExperimentEntity
	if err := d.DB.WithContext(ctx).Where("experiment_id = ?", experimentID).Last(&dbExp).Error; err != nil {
		return nil, err
	}
	exp := toPreparedExperimentDomain(&dbExp)
	return &exp, nil
}

func toPreparedExperimentDomain(dbExp *dbEntity.PreparedExperimentEntity) entity.PreparedExperiment {
	return entity.PreparedExperiment{
		ID:           dbExp.ID,
		UserID:       dbExp.UserID,
		ExperimentID: dbExp.ExperimentID,
		CropAlt:      dbExp.CropAlt,
		BGRType:      entity.BGRType(dbExp.BGRType),
		BGRAlt:       dbExp.BGRAlt,
		PathToData:   dbExp.PathToData,
		Status:       entity.PreparedExperimentStatus(dbExp.Status),
		ErrorMsg:     dbExp.ErrorMsg,
	}
}
