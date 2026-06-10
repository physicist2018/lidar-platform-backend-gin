package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/redis/go-redis/v9"

	"github.com/go-playground/validator/v10"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/controller"
	"github.com/kshmirko/lidar-platform-go/internal/delivery/http/route"
	usecaseImpl "github.com/kshmirko/lidar-platform-go/internal/domain/usecase/implementation"
	cacheImpl "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/cache/implementation"
	dsImpl "github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance/implementation"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/db"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/queue"
	repoImpl "github.com/kshmirko/lidar-platform-go/internal/infrastructure/repository"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/internal/utils/auth"
	"github.com/kshmirko/lidar-platform-go/internal/utils/worker"
	"github.com/kshmirko/lidar-platform-go/pkg/dto"
)

type BootstrapConfig struct {
	DB              *gorm.DB
	Redis           *redis.Client
	Log             *logrus.Logger
	CacheTTLDefault time.Duration
	EchoEngine      *echo.Echo
	WorkerPool      *worker.Pool // kept for transition period
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

	// Legacy worker pool (kept for transition period)
	maxWorkers := cfg.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	workerPool := worker.NewPool(maxWorkers, log)
	workerPool.Start(maxWorkers)

	// Asynq client & task store
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}
	queueClient := queue.NewClient(redisOpt, log)
	taskStore := queue.NewTaskStore(redisConn)

	echoEngine := echo.New()
	echoEngine.Use(echootel.NewMiddleware("lidar-platform"))

	// Centralized HTTP error handler
	echoEngine.HTTPErrorHandler = func(c *echo.Context, err error) {
		code := http.StatusInternalServerError
		msg := "internal error"
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = he.Message
		} else if codeErr, ok := err.(interface{ StatusCode() int }); ok {
			code = codeErr.StatusCode()
			msg = err.Error()
		} else if err != nil {
			msg = err.Error()
		}
		// Default error handler — always send a JSON error response
		c.JSON(code, dto.ErrorResponse{Error: msg})
	}

	// Register validator
	validate := validator.New()
	echoEngine.Validator = &CustomValidator{validator: validate}

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
	getExpChannelsUC := usecaseImpl.NewGetExperimentChannelsUseCaseImpl(expRepo, log)

	// --- Wire PreparedExperiment domain ---
	prepDataSource := dsImpl.NewPreparedExperimentDataSourceImpl(dbConn, log)
	prepRepo := repoImpl.NewPreparedExperimentRepositoryImpl(prepDataSource, log)
	// Use asynq-based use cases
	prepareExpUC := usecaseImpl.NewPrepareExperimentUseCaseImpl(expRepo, prepRepo, queueClient, log)
	visualizePrepUC := usecaseImpl.NewVisualizePreparedExperimentUseCaseImpl(queueClient, log)
	gluePrepUC := usecaseImpl.NewGluePreparedExperimentUseCaseImpl(prepRepo, queueClient, log)

	expController := controller.NewExperimentController(log, createExpUC, getExpByIDUC, getAllExpUC, getExpChannelsUC, prepareExpUC, visualizePrepUC, gluePrepUC, taskStore)

	route.NewRouteConfig(echoEngine, cfg.JWTSecret, userController, authController, expController).Setup()

	return &BootstrapConfig{
		DB:              dbConn,
		Redis:           redisConn,
		Log:             log,
		CacheTTLDefault: cfg.CacheTTLDefault,
		EchoEngine:      echoEngine,
		WorkerPool:      workerPool,
	}, nil
}

// CustomValidator wraps go-playground/validator for Echo v5.
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}
