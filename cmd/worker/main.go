package main

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/config"
	"github.com/physicist2018/lidar-platform-go/internal/domain/processing"
	dsImpl "github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance/implementation"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/db"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/queue"
	repoImpl "github.com/physicist2018/lidar-platform-go/internal/infrastructure/repository"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/storage"
)

func main() {
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	dbConn, err := db.NewPostgresDB(cfg.DBSource)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	redisConn, err := db.NewRedisClient(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	minioClient, err := storage.NewMinioClient(
		cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey,
		cfg.MinioBucket, cfg.MinioUseSSL, logger,
	)
	if err != nil {
		log.Fatalf("failed to initialize minio: %v", err)
	}

	// Repositories
	prepRepo := repoImpl.NewPreparedExperimentRepositoryImpl(
		dsImpl.NewPreparedExperimentDataSourceImpl(dbConn, logger), logger,
	)
	chartRepo := repoImpl.NewExperimentChartRepositoryImpl(
		dsImpl.NewExperimentChartDataSourceImpl(dbConn, logger), logger,
	)

	// --- Processing domain repos ---
	lidarPackDS := dsImpl.NewLidarPackDataSourceImpl(dbConn, logger)
	lidarPackRepo := repoImpl.NewLidarPackRepositoryImpl(lidarPackDS, logger)

	expDS := dsImpl.NewExperimentDataSourceImpl(dbConn, logger)
	expRepo := repoImpl.NewExperimentRepositoryImpl(expDS, logger)

	procRunDS := dsImpl.NewProcessingRunDataSourceImpl(dbConn, logger)
	procRunRepo := repoImpl.NewProcessingRunRepositoryImpl(procRunDS, logger)

	procSigDS := dsImpl.NewProcessedSignalDataSourceImpl(dbConn, logger)
	procSigRepo := repoImpl.NewProcessedSignalRepositoryImpl(procSigDS, logger)

	// Register algorithm processors
	processorReg := processing.NewRegistry()
	stage0 := processing.NewStage0Processor(lidarPackRepo, procSigRepo, expRepo, logger)
	processorReg.Register(stage0)

	// Task store (for polling results)
	taskStore := queue.NewTaskStore(redisConn)

	// Handler dependencies
	deps := &queue.HandlerDeps{
		PrepRepo:      prepRepo,
		ChartRepo:     chartRepo,
		ProcRunRepo:   procRunRepo,
		ProcSigRepo:   procSigRepo,
		ExpRepo:       expRepo,
		LidarPackRepo: lidarPackRepo,
		ProcessorReg:  processorReg,
		Minio:         minioClient,
		TaskStore:     taskStore,
		Log:           logger,
	}

	// Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.RedisAddress,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		},
		asynq.Config{
			Concurrency: cfg.MaxWorkers,
			Logger:      logger,
		},
	)

	mux := queue.NewServeMux(deps)

	logger.WithFields(logrus.Fields{
		"redis_addr":  cfg.RedisAddress,
		"concurrency": cfg.MaxWorkers,
	}).Info("starting asynq worker")

	if err := srv.Run(mux); err != nil {
		log.Fatalf("asynq server error: %v", err)
	}
}
