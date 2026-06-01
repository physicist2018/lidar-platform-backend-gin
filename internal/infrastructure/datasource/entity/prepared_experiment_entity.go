package entity

import (
	"time"

	"gorm.io/gorm"
)

type PreparedExperimentEntity struct {
	ID           uint    `gorm:"primaryKey"`
	UserID       uint    `gorm:"not null;default:1;index"`
	ExperimentID uint    `gorm:"not null;index"`
	CropAlt      float64 `gorm:"not null"`
	BGRType      string  `gorm:"size:20;not null"`
	BGRAlt       float64 `gorm:"default:0"`
	PathToData   string  `gorm:"size:500"`
	Status       string  `gorm:"size:20;not null;default:staged"`
	ErrorMsg     string  `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (PreparedExperimentEntity) TableName() string { return "prepared_experiments" }
