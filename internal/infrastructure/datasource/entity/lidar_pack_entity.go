package entity

import "time"

type LidarPackEntity struct {
	ID           uint              `gorm:"primaryKey"`
	ExperimentID uint              `gorm:"not null;index"`
	StartTime    *time.Time        `gorm:"default:null"`
	StopTime     *time.Time        `gorm:"default:null"`
	CreatedAt    time.Time         `gorm:"autoCreateTime"`
	Files        []LidarFileEntity `gorm:"foreignKey:PackID;constraint:OnDelete:CASCADE"`
}

func (LidarPackEntity) TableName() string { return "lidar_packs" }
