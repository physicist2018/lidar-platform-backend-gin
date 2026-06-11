package entity

import "gorm.io/datatypes"

type LidarProfileEntity struct {
	ID           uint           `gorm:"primaryKey"`
	FileID       uint           `gorm:"not null;index"`
	Active       bool           `gorm:"default:false"`
	IsPhoton     bool           `gorm:"default:false"`
	LaserType    int            `gorm:"default:0"`
	NDataPoints  int            `gorm:"default:0"`
	Reserved     datatypes.JSON `gorm:"type:jsonb;default:'[]'"`
	HighVoltage  int            `gorm:"default:0"`
	BinWidth     float64        `gorm:"default:0"`
	Wavelength   float64        `gorm:"default:0;index"`
	Polarization string         `gorm:"size:50;default:''"`
	BinShift     int            `gorm:"default:0"`
	DecBinShift  int            `gorm:"default:0"`
	AdcBits      int            `gorm:"default:0"`
	NShots       int            `gorm:"default:0"`
	DiscrLevel   float64        `gorm:"default:0"`
	DeviceID     string         `gorm:"size:255;default:'';index"`
	NCrate       int            `gorm:"default:0"`
	Signal       []byte         `gorm:"type:bytea"`
}

func (LidarProfileEntity) TableName() string { return "lidar_profiles" }
