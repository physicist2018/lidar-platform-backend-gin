package implementation

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
)

type getExperimentChannelsUseCaseImpl struct {
	repo repository.ExperimentRepository
	log  *logrus.Logger
}

var _ usecase.GetExperimentChannelsUseCase = (*getExperimentChannelsUseCaseImpl)(nil)

func NewGetExperimentChannelsUseCaseImpl(
	repo repository.ExperimentRepository,
	log *logrus.Logger,
) *getExperimentChannelsUseCaseImpl {
	return &getExperimentChannelsUseCaseImpl{repo: repo, log: log}
}

func (u *getExperimentChannelsUseCaseImpl) Execute(ctx context.Context, id uint) ([]entity.ExperimentChannel, error) {
	tracer := otel.Tracer("usecase")
	ctx, span := tracer.Start(ctx, "GetExperimentChannelsUseCase.Execute")
	defer span.End()

	start := time.Now()
	span.SetAttributes(attribute.Int("experiment_id", int(id)))

	exp, err := u.repo.FindByID(ctx, id)
	if err != nil {
		u.log.WithFields(logrus.Fields{
			"operation": "GetExperimentChannelsUseCase.Execute",
			"duration":  time.Since(start).String(),
			"error":     err,
		}).Error("failed to get experiment")
		return nil, fmt.Errorf("get experiment channels: %w", err)
	}

	if exp.AvailableChannels == nil {
		return []entity.ExperimentChannel{}, nil
	}

	u.log.WithFields(logrus.Fields{
		"operation":     "GetExperimentChannelsUseCase.Execute",
		"duration":      time.Since(start).String(),
		"channel_count": len(exp.AvailableChannels),
	}).Info("get experiment channels success")

	return exp.AvailableChannels, nil
}
