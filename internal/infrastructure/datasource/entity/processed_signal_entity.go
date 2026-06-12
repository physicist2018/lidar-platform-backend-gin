package entity

import (
	"github.com/physicist2018/lidar-platform-go/internal/utils/gorm/datatypes"
)

type ProcessedSignalEntity struct {
	ID                uint                   `gorm:"primaryKey"`
	ProcessingRunID   uint                   `gorm:"not null;index"`
	OriginalProfileID uint                   `gorm:"not null;index"`
	Wavelength        float64                `gorm:"not null"`
	Polarization      string                 `gorm:"size:10;not null"`
	IsPhoton          bool                   `gorm:"not null"`
	Signal            datatypes.Float64Slice `gorm:"type:bytea;not null"`
}

func (ProcessedSignalEntity) TableName() string { return "processed_signals" }
