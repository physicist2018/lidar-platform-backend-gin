package main

import (
	"log"

	"github.com/kshmirko/lidar-platform-go/internal/config"
	dbEntity "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/entity"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/db"
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

	log.Println("Running auto-migration...")
	if err := dbConn.AutoMigrate(
		&dbEntity.UserEntity{},
		&dbEntity.ExperimentEntity{},
	); err != nil {
		log.Fatalf("auto-migration failed: %v", err)
	}

	log.Println("Migration completed successfully: users, experiments")
}
