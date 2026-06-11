package entity

import "time"

type LidarFileEntity struct {
	ID           uint                 `gorm:"primaryKey"`
	PackID       uint                 `gorm:"not null;index"`
	Filename     string               `gorm:"size:500;not null"`
	Site         string               `gorm:"size:255;default:''"`
	StartTime    *time.Time           `gorm:"default:null;index"`
	StopTime     *time.Time           `gorm:"default:null"`
	Altitude     float64              `gorm:"default:0"`
	Longitude    float64              `gorm:"default:0"`
	Latitude     float64              `gorm:"default:0"`
	Zenith       float64              `gorm:"default:0"`
	Laser1NShots int                  `gorm:"default:0"`
	Laser1Freq   int                  `gorm:"default:0"`
	Laser2NShots int                  `gorm:"default:0"`
	Laser2Freq   int                  `gorm:"default:0"`
	Laser3NShots int                  `gorm:"default:0"`
	Laser3Freq   int                  `gorm:"default:0"`
	NDatasets    int                  `gorm:"default:0"`
	Profiles     []LidarProfileEntity `gorm:"foreignKey:FileID;constraint:OnDelete:CASCADE"`
}

func (LidarFileEntity) TableName() string { return "lidar_files" }
