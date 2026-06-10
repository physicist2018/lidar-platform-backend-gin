package main

import (
	"log"

	"github.com/kshmirko/lidar-platform-go/internal/config"

	_ "github.com/kshmirko/lidar-platform-go/docs"
)

// @title						Lidar Platform API
// @version					1.0
// @description				REST API for the Lidar Platform — user management, experiment processing.
// @contact.name				API Support
// @contact.email				kshmirko@dvo.ru
// @license.name				MIT
// @license.url				https://opensource.org/licenses/MIT
// @host						localhost:8080
// @BasePath					/
//
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				JWT Bearer token. Example: "Bearer eyJhbGciOiJIUzI1NiIs..."
func main() {
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	boot, err := config.Initialize(cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	if err := boot.EchoEngine.Start(cfg.ServerAddress); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
