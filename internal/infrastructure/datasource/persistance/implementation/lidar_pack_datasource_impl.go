package implementation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/kshmirko/lidar-platform-go/internal/utils/licel"
)

var _ persistance.LidarPackDataSource = (*LidarPackDataSourceImpl)(nil)

type LidarPackDataSourceImpl struct {
	DB  *gorm.DB
	Log *logrus.Logger
}

func NewLidarPackDataSourceImpl(db *gorm.DB, log *logrus.Logger) *LidarPackDataSourceImpl {
	return &LidarPackDataSourceImpl{DB: db, Log: log}
}

// SavePack saves the complete hierarchy (lidar_pack → lidar_files → lidar_profiles)
// in a single transaction. FileIDs and ProfileIDs are populated back.
func (d *LidarPackDataSourceImpl) SavePack(ctx context.Context, pack *entity.LidarPack) error {
	packEntity := &dbEntity.LidarPackEntity{
		ExperimentID: pack.ExperimentID,
		PackType:     pack.PackType,
		StartTime:    pack.StartTime,
		StopTime:     pack.StopTime,
	}

	err := d.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create the pack
		if err := tx.Create(packEntity).Error; err != nil {
			return fmt.Errorf("create lidar_pack: %w", err)
		}
		pack.ID = packEntity.ID

		// 2. Create each file with profiles
		for fi := range pack.Files {
			df := &pack.Files[fi]
			fileEntity := toLidarFileEntity(packEntity.ID, df)
			if err := tx.Create(fileEntity).Error; err != nil {
				return fmt.Errorf("create lidar_file %q: %w", df.Filename, err)
			}
			df.ID = fileEntity.ID

			for pi := range df.Profiles {
				dp := &df.Profiles[pi]
				profEntity := toLidarProfileEntity(fileEntity.ID, dp)
				if err := tx.Create(profEntity).Error; err != nil {
					return fmt.Errorf("create lidar_profile for file %d: %w", fileEntity.ID, err)
				}
				dp.ID = profEntity.ID
			}
		}

		return nil
	})
	if err != nil {
		d.Log.WithError(err).Error("LidarPackDataSource.SavePack failed")
		return err
	}

	d.Log.WithFields(logrus.Fields{
		"pack_id": pack.ID,
		"files":   len(pack.Files),
	}).Info("lidar pack saved successfully")

	return nil
}

// toLidarFileEntity converts domain LidarFile to GORM entity.
func toLidarFileEntity(packID uint, df *entity.LidarFile) *dbEntity.LidarFileEntity {
	return &dbEntity.LidarFileEntity{
		PackID:       packID,
		Filename:     df.Filename,
		Site:         df.Site,
		StartTime:    df.StartTime,
		StopTime:     df.StopTime,
		Altitude:     df.Altitude,
		Longitude:    df.Longitude,
		Latitude:     df.Latitude,
		Zenith:       df.Zenith,
		Laser1NShots: df.Laser1NShots,
		Laser1Freq:   df.Laser1Freq,
		Laser2NShots: df.Laser2NShots,
		Laser2Freq:   df.Laser2Freq,
		Laser3NShots: df.Laser3NShots,
		Laser3Freq:   df.Laser3Freq,
		NDatasets:    df.NDatasets,
	}
}

// toLidarProfileEntity converts domain LidarProfile to GORM entity.
func toLidarProfileEntity(fileID uint, dp *entity.LidarProfile) *dbEntity.LidarProfileEntity {
	reservedJSON, _ := json.Marshal(dp.Reserved)
	if dp.Reserved == nil {
		reservedJSON = json.RawMessage("[]")
	}
	return &dbEntity.LidarProfileEntity{
		FileID:       fileID,
		Active:       dp.Active,
		IsPhoton:     dp.IsPhoton,
		LaserType:    dp.LaserType,
		NDataPoints:  dp.NDataPoints,
		Reserved:     reservedJSON,
		HighVoltage:  dp.HighVoltage,
		BinWidth:     dp.BinWidth,
		Wavelength:   dp.Wavelength,
		Polarization: dp.Polarization,
		BinShift:     dp.BinShift,
		DecBinShift:  dp.DecBinShift,
		AdcBits:      dp.AdcBits,
		NShots:       dp.NShots,
		DiscrLevel:   dp.DiscrLevel,
		DeviceID:     dp.DeviceID,
		NCrate:       dp.NCrate,
		Signal:       licel.Float64sToBytes(dp.Signal),
	}
}
