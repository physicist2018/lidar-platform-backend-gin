package config

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/controller"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/route"
	usecaseImpl "github.com/kshmirko/lidar-platform-go/internal/domain/usecase/implementation"
	cacheImpl "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/cache/implementation"
	dsImpl "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance/implementation"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/db"
	repoImpl "github.com/kshmirko/lidar-platform-go/internal/infrastructure/repository"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/internal/utils/auth"
	"github.com/kshmirko/lidar-platform-go/internal/utils/worker"
)

type BootstrapConfig struct {
	DB              *gorm.DB
	Redis           *redis.Client
	Log             *logrus.Logger
	CacheTTLDefault time.Duration
	GinEngine       *gin.Engine
	WorkerPool      *worker.Pool
}

// Initialize builds the full dependency graph.
func Initialize(cfg *Config) (*BootstrapConfig, error) {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	dbConn, err := db.NewPostgresDB(cfg.DBSource)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres: %w", err)
	}

	redisConn, err := db.NewRedisClient(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		return nil, fmt.Errorf("initialize redis: %w", err)
	}

	// Minio client
	minioClient, err := storage.NewMinioClient(
		cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey,
		cfg.MinioBucket, cfg.MinioUseSSL, log,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize minio: %w", err)
	}

	// Worker pool
	maxWorkers := cfg.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	workerPool := worker.NewPool(maxWorkers, log)
	workerPool.Start(maxWorkers)

	ginEngine := gin.Default()
	ginEngine.Use(otelgin.Middleware("lidar-platform"))

	// --- JWT Config ---
	jwtConfig := auth.JWTConfig{
		Secret:     cfg.JWTSecret,
		Expiration: cfg.JWTExpiration,
	}

	// --- Wire User domain ---
	userDataSource := dsImpl.NewUserDataSourceImpl(dbConn, log)
	userCache := cacheImpl.NewUserCacheImpl(redisConn, cfg.CacheTTLDefault, log)
	userRepo := repoImpl.NewUserRepositoryImpl(userDataSource, userCache, log)

	getAllUsersUC := usecaseImpl.NewGetAllUsersUseCaseImpl(userRepo, log)
	getUserByIDUC := usecaseImpl.NewGetUserByIDUseCaseImpl(userRepo, log)
	createUserUC := usecaseImpl.NewCreateUserUseCaseImpl(userRepo, log)
	updateUserUC := usecaseImpl.NewUpdateUserUseCaseImpl(userRepo, log)
	deleteUserUC := usecaseImpl.NewDeleteUserUseCaseImpl(userRepo, log)
	loginUC := usecaseImpl.NewLoginUseCaseImpl(userRepo, jwtConfig, log)

	userController := controller.NewUserController(
		log, getAllUsersUC, getUserByIDUC, createUserUC, updateUserUC, deleteUserUC,
	)
	authController := controller.NewAuthController(log, loginUC)

	// --- Wire Experiment domain ---
	expDataSource := dsImpl.NewExperimentDataSourceImpl(dbConn, log)
	expRepo := repoImpl.NewExperimentRepositoryImpl(expDataSource, log)

	createExpUC := usecaseImpl.NewCreateExperimentUseCaseImpl(expRepo, minioClient, workerPool, log)
	getExpByIDUC := usecaseImpl.NewGetExperimentByIDUseCaseImpl(expRepo, log)
	getAllExpUC := usecaseImpl.NewGetAllExperimentsUseCaseImpl(expRepo, log)

	expController := controller.NewExperimentController(log, createExpUC, getExpByIDUC, getAllExpUC)

	route.NewRouteConfig(ginEngine, cfg.JWTSecret, userController, authController, expController).Setup()

	return &BootstrapConfig{
		DB:              dbConn,
		Redis:           redisConn,
		Log:             log,
		CacheTTLDefault: cfg.CacheTTLDefault,
		GinEngine:       ginEngine,
		WorkerPool:      workerPool,
	}, nil
}
