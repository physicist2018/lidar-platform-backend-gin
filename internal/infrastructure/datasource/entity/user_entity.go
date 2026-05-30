package entity

import (
	"time"

	"gorm.io/gorm"
)

type UserEntity struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100;not null"`
	Email     string `gorm:"size:255;uniqueIndex;not null"`
	Role      string `gorm:"size:20;not null;default:guest"`
	Password  string `gorm:"size:255;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (UserEntity) TableName() string { return "users" }
