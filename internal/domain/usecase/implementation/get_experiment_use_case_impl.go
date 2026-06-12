package implementation

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type getExperimentByIDUseCaseImpl struct {
	repo repository.ExperimentRepository
	log  *logrus.Logger
}

var _ usecase.GetExperimentByIDUseCase = (*getExperimentByIDUseCaseImpl)(nil)

func NewGetExperimentByIDUseCaseImpl(repo repository.ExperimentRepository, log *logrus.Logger) *getExperimentByIDUseCaseImpl {
	return &getExperimentByIDUseCaseImpl{repo: repo, log: log}
}

func (u *getExperimentByIDUseCaseImpl) Execute(ctx context.Context, id uint) (*entity.Experiment, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "GetExperimentByIDUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(attribute.Int("experiment_id", int(id)))

	exp, err := u.repo.FindByID(ctx, id)
	if err != nil {
		u.log.WithFields(logrus.Fields{
			"operation": "GetExperimentByIDUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to get experiment")
		return nil, err
	}

	u.log.WithFields(logrus.Fields{
		"operation": "GetExperimentByIDUseCase.Execute",
		"duration":  time.Since(start).String(),
	}).Info("get experiment by id success")

	return exp, nil
}

// ---

type getAllExperimentsUseCaseImpl struct {
	repo repository.ExperimentRepository
	log  *logrus.Logger
}

var _ usecase.GetAllExperimentsUseCase = (*getAllExperimentsUseCaseImpl)(nil)

func NewGetAllExperimentsUseCaseImpl(repo repository.ExperimentRepository, log *logrus.Logger) *getAllExperimentsUseCaseImpl {
	return &getAllExperimentsUseCaseImpl{repo: repo, log: log}
}

func (u *getAllExperimentsUseCaseImpl) Execute(ctx context.Context, filter *entity.ExperimentFilter) (*pagination.Pagination[entity.Experiment], error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "GetAllExperimentsUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(
		attribute.Int("page", filter.Page),
		attribute.Int("limit", filter.Limit),
	)

	result, err := u.repo.FindAll(ctx, filter)
	if err != nil {
		u.log.WithFields(logrus.Fields{
			"operation": "GetAllExperimentsUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to get all experiments")
		return nil, err
	}

	u.log.WithFields(logrus.Fields{
		"operation": "GetAllExperimentsUseCase.Execute",
		"duration":  time.Since(start).String(),
		"count":     len(result.Data),
	}).Info("get all experiments success")

	return result, nil
}
