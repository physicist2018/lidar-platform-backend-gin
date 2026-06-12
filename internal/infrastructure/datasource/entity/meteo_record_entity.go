package entity

import (
	"github.com/physicist2018/lidar-platform-go/internal/utils/meteo"
)

// MeteoRecordEntity stores all meteo levels for an experiment as binary bytea arrays.
type MeteoRecordEntity struct {
	ID           uint               `gorm:"primaryKey"`
	ExperimentID uint               `gorm:"not null;index"`
	Pres         meteo.Float64Slice `gorm:"type:bytea;not null"`
	Hght         meteo.Float64Slice `gorm:"type:bytea;not null"`
	Temp         meteo.Float64Slice `gorm:"type:bytea;not null"`
	Relh         meteo.Float64Slice `gorm:"type:bytea"`
	Mixr         meteo.Float64Slice `gorm:"type:bytea"`
	Drct         meteo.Float64Slice `gorm:"type:bytea"`
	Sknt         meteo.Float64Slice `gorm:"type:bytea"`
}

func (MeteoRecordEntity) TableName() string { return "meteo_records" }
