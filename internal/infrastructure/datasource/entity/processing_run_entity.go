package entity

import (
	"time"

	"gorm.io/gorm"
)

type ProcessingRunEntity struct {
	ID           uint   `gorm:"primaryKey"`
	ExperimentID uint   `gorm:"not null;index"`
	UserID       uint   `gorm:"not null;default:1;index"`
	Algorithm    string `gorm:"size:50;not null"`
	Params       []byte `gorm:"type:jsonb;not null"`
	DependsOnID  *uint  `gorm:"default:null;index"`
	Status       string `gorm:"size:20;not null;default:staged"`
	ErrorMsg     string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (ProcessingRunEntity) TableName() string { return "processing_runs" }
