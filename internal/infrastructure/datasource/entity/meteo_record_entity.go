package entity

import (
	"github.com/physicist2018/lidar-platform-go/internal/utils/gorm/datatypes"
)

// MeteoRecordEntity stores all meteo levels for an experiment as binary bytea arrays.
type MeteoRecordEntity struct {
	ID           uint                   `gorm:"primaryKey"`
	ExperimentID uint                   `gorm:"not null;index"`
	Pres         datatypes.Float64Slice `gorm:"type:bytea;not null"`
	Hght         datatypes.Float64Slice `gorm:"type:bytea;not null"`
	Temp         datatypes.Float64Slice `gorm:"type:bytea;not null"`
	Relh         datatypes.Float64Slice `gorm:"type:bytea"`
	Mixr         datatypes.Float64Slice `gorm:"type:bytea"`
	Drct         datatypes.Float64Slice `gorm:"type:bytea"`
	Sknt         datatypes.Float64Slice `gorm:"type:bytea"`
}

func (MeteoRecordEntity) TableName() string { return "meteo_records" }
