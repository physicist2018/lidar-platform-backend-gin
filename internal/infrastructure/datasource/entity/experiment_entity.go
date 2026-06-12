package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ExperimentEntity struct {
	ID                   uint           `gorm:"primaryKey"`
	UserID               uint           `gorm:"not null;default:1;index"`
	Title                string         `gorm:"size:255;not null"`
	Comments             string         `gorm:"type:text"`
	MeasurementStartTime *time.Time     `gorm:"default:null"`
	MeasurementStopTime  *time.Time     `gorm:"default:null"`
	LidarPackID          *uint          `gorm:"default:null;index"`
	BgrFileID            *uint          `gorm:"default:null;index"`
	LicelZipPath         string         `gorm:"size:500"`
	LicelBgrPath         string         `gorm:"size:500"`
	MeteoFilePath        string         `gorm:"size:500"`
	Status               string         `gorm:"size:20;not null;default:staged"`
	ErrorMsg             string         `gorm:"type:text"`
	AvailableChannels    datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (ExperimentEntity) TableName() string { return "experiments" }
