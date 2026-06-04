package entity

import (
	"time"

	"gorm.io/gorm"
)

type ExperimentChartEntity struct {
	ID           uint    `gorm:"primaryKey"`
	ExperimentID uint    `gorm:"not null;index"`
	ChartType    string  `gorm:"size:10;not null"`
	Formula      string  `gorm:"size:20;not null"`
	Wavelen      float64 `gorm:"not null"`
	Polarization string  `gorm:"size:50;not null"`
	IsPhoton     int8    `gorm:"not null"`
	Glued        int8    `gorm:"not null;default:0"`
	PathToObject string  `gorm:"size:500;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (ExperimentChartEntity) TableName() string { return "experiment_charts" }
