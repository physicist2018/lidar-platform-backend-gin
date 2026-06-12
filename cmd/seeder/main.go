package main

import (
	"log"

	"github.com/physicist2018/lidar-platform-go/internal/config"
	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	dbEntity "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/db"
	"github.com/physicist2018/lidar-platform-go/internal/utils/hash"
)

func main() {
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbConn, err := db.NewPostgresDB(cfg.DBSource)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// AutoMigrate to ensure tables exist
	if err := dbConn.AutoMigrate(&dbEntity.UserEntity{}); err != nil {
		log.Fatalf("auto-migration failed: %v", err)
	}

	// Check if admin already exists
	var count int64
	dbConn.Model(&dbEntity.UserEntity{}).Where("email = ?", "admin@lidar-platform.io").Count(&count)
	if count > 0 {
		log.Println("Admin user already exists, skipping.")
		return
	}

	// Hash password
	hashedPassword, err := hash.Password("admin2332361")
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	// Create admin
	admin := dbEntity.UserEntity{
		Name:     "Administrator",
		Email:    "kshmirko@dvo.ru",
		Role:     string(entity.RoleAdmin),
		Password: hashedPassword,
	}

	if err := dbConn.Create(&admin).Error; err != nil {
		log.Fatalf("failed to create admin user: %v", err)
	}

	log.Println("Admin user created successfully!")
	log.Printf("   Email:    %s\n", admin.Email)
	log.Printf("   Password: %s\n", "admin2332361") //nolint:gosec
	log.Println("   Role:     admin")
}
