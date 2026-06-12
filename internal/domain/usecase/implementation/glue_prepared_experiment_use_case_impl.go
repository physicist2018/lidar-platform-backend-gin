package implementation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/queue"
)

type gluePreparedExperimentUseCaseImpl struct {
	prepRepo    repository.PreparedExperimentRepository
	queueClient *queue.Client
	log         *logrus.Logger
}

var _ usecase.GluePreparedExperimentUseCase = (*gluePreparedExperimentUseCaseImpl)(nil)

func NewGluePreparedExperimentUseCaseImpl(
	prepRepo repository.PreparedExperimentRepository,
	queueClient *queue.Client,
	log *logrus.Logger,
) *gluePreparedExperimentUseCaseImpl {
	return &gluePreparedExperimentUseCaseImpl{
		prepRepo:    prepRepo,
		queueClient: queueClient,
		log:         log,
	}
}

func (u *gluePreparedExperimentUseCaseImpl) Execute(
	ctx context.Context,
	experimentID uint,
	wavelengths []float64,
	polarization string,
	h1, h2 float64,
) error {
	// Find PreparedExperiment by ExperimentID
	prep, err := u.prepRepo.FindByExperimentID(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("prepared experiment not found for experiment %d: %w", experimentID, err)
	}

	// Validate status
	if prep.Status != entity.PrepStatusDoneStageOne && prep.Status != entity.PrepStatusDoneStageTwo {
		return fmt.Errorf(
			"data must be in PrepStatusDoneStageOne or PrepStatusDoneStageTwo status, current: %s",
			prep.Status,
		)
	}

	// Enqueue async task via asynq
	payload := queue.GluePayload{
		PrepID:       prep.ID,
		ExperimentID: experimentID,
		PathToData:   prep.PathToData,
		Wavelengths:  wavelengths,
		Polarization: polarization,
		H1:           h1,
		H2:           h2,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal glue payload: %w", err)
	}

	task := asynq.NewTask(queue.TypeGlue, data)
	_, err = u.queueClient.Enqueue(task)
	if err != nil {
		return fmt.Errorf("enqueue glue task: %w", err)
	}

	u.log.WithFields(logrus.Fields{
		"prepared_experiment_id": prep.ID,
		"experiment_id":          experimentID,
		"wavelengths":            wavelengths,
		"polarization":           polarization,
		"h1":                     h1,
		"h2":                     h2,
	}).Info("glue task enqueued via asynq")

	return nil
}
