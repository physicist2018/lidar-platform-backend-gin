package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/physicist2018/lidar-platform-go/internal/config"

	_ "github.com/physicist2018/lidar-platform-go/docs"
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

	// Start HTTP server in a goroutine
	go func() {
		boot.Log.WithField("address", cfg.ServerAddress).Info("server starting")
		if err := boot.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			boot.Log.WithError(err).Fatal("server error")
		}
	}()

	// Wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	boot.Log.WithField("signal", sig.String()).Info("received shutdown signal")

	// Graceful shutdown with 30s timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := boot.Shutdown(shutdownCtx); err != nil {
		boot.Log.WithError(err).Error("forced shutdown")
		os.Exit(1)
	}
}
