package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

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
)

type BootstrapConfig struct {
	DB              *gorm.DB
	Redis           *redis.Client
	Log             *logrus.Logger
	CacheTTLDefault time.Duration
	Router          *chi.Mux
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

	// --- Chi Router ---
	router := chi.NewRouter()

	// Middleware
	router.Use(chiMiddleware.RequestID)
	router.Use(chiMiddleware.RealIP)
	router.Use(chiMiddleware.Logger)
	// Centralized panic recovery handler (returns JSON, logged via logrus)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Errorf("panic recovered: %v", rec)
					http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})

	// Request logging middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.WithFields(logrus.Fields{
				"method": r.Method,
				"uri":    r.URL.String(),
			}).Info("request")
			next.ServeHTTP(w, r)
		})
	})

	// Register validator
	validate := validator.New()

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
		log, getAllUsersUC, getUserByIDUC, createUserUC, updateUserUC, deleteUserUC, validate,
	)
	authController := controller.NewAuthController(log, loginUC, validate)

	// --- LidarPack DataSource & Repository ---
	lidarPackDataSource := dsImpl.NewLidarPackDataSourceImpl(dbConn, log)
	lidarPackRepo := repoImpl.NewLidarPackRepositoryImpl(lidarPackDataSource, log)

	// --- Wire Experiment domain ---
	expDataSource := dsImpl.NewExperimentDataSourceImpl(dbConn, log)
	expRepo := repoImpl.NewExperimentRepositoryImpl(expDataSource, log)

	createExpUC := usecaseImpl.NewCreateExperimentUseCaseImpl(expRepo, lidarPackRepo, minioClient, workerPool, log)
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

	expController := controller.NewExperimentController(log, createExpUC, getExpByIDUC, getAllExpUC, getExpChannelsUC, prepareExpUC, visualizePrepUC, gluePrepUC, taskStore, validate)

	route.NewRouteConfig(router, cfg.JWTSecret, userController, authController, expController).Setup()

	return &BootstrapConfig{
		DB:              dbConn,
		Redis:           redisConn,
		Log:             log,
		CacheTTLDefault: cfg.CacheTTLDefault,
		Router:          router,
		WorkerPool:      workerPool,
	}, nil
}
