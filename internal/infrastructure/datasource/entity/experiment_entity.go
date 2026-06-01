package entity

import (
	"time"

	"gorm.io/gorm"
)

type ExperimentEntity struct {
	ID                   uint       `gorm:"primaryKey"`
	UserID               uint       `gorm:"not null;index"`
	Title                string     `gorm:"size:255;not null"`
	Comments             string     `gorm:"type:text"`
	MeasurementStartTime *time.Time `gorm:"default:null"`
	MeasurementStopTime  *time.Time `gorm:"default:null"`
	LicelZipPath         string     `gorm:"size:500"`
	LicelBgrPath         string     `gorm:"size:500"`
	MeteoFilePath        string     `gorm:"size:500"`
	Status               string     `gorm:"size:20;not null;default:staged"`
	ErrorMsg             string     `gorm:"type:text"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (ExperimentEntity) TableName() string { return "experiments" }
