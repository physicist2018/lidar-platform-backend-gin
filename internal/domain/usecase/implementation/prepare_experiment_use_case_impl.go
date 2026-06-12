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

type prepareExperimentUseCaseImpl struct {
	expRepo     repository.ExperimentRepository
	prepRepo    repository.PreparedExperimentRepository
	queueClient *queue.Client
	log         *logrus.Logger
}

var _ usecase.PrepareExperimentUseCase = (*prepareExperimentUseCaseImpl)(nil)

func NewPrepareExperimentUseCaseImpl(
	expRepo repository.ExperimentRepository,
	prepRepo repository.PreparedExperimentRepository,
	queueClient *queue.Client,
	log *logrus.Logger,
) *prepareExperimentUseCaseImpl {
	return &prepareExperimentUseCaseImpl{
		expRepo:     expRepo,
		prepRepo:    prepRepo,
		queueClient: queueClient,
		log:         log,
	}
}

func (u *prepareExperimentUseCaseImpl) Execute(
	ctx context.Context,
	userID, experimentID uint,
	cropAlt float64,
	bgrType entity.BGRType,
	bgrAlt float64,
) (*entity.PreparedExperiment, error) {
	// Validate that the experiment exists
	exp, err := u.expRepo.FindByID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("experiment not found: %w", err)
	}

	if exp.Status != entity.StatusDone {
		return nil, fmt.Errorf("experiment %d must be in 'done' status, current: %s", experimentID, exp.Status)
	}

	// Build the target Minio path for prepared data
	pathToData := fmt.Sprintf("experiments/%d/prepared/licel-prepared.zip", experimentID)

	prep := &entity.PreparedExperiment{
		UserID:       userID,
		ExperimentID: experimentID,
		CropAlt:      cropAlt,
		BGRType:      bgrType,
		BGRAlt:       bgrAlt,
		PathToData:   pathToData,
		Status:       entity.PrepStatusStaged,
	}

	if err := prep.ValidateBGRParams(); err != nil {
		return nil, err
	}

	if err := u.prepRepo.Create(ctx, prep); err != nil {
		return nil, fmt.Errorf("create prepared experiment: %w", err)
	}

	// Enqueue async task via asynq
	payload := queue.PreparePayload{
		PrepID:       prep.ID,
		ExperimentID: experimentID,
		CropAlt:      cropAlt,
		BGRType:      string(bgrType),
		BGRAlt:       bgrAlt,
		PathToData:   pathToData,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal prepare payload: %w", err)
	}

	task := asynq.NewTask(queue.TypePrepare, data)
	_, err = u.queueClient.Enqueue(task)
	if err != nil {
		// If enqueue fails, mark the prepared experiment as failed
		_ = u.prepRepo.Update(ctx, &entity.PreparedExperiment{
			ID:       prep.ID,
			Status:   entity.PrepStatusFailed,
			ErrorMsg: fmt.Sprintf("failed to enqueue task: %s", err.Error()),
		})
		return nil, fmt.Errorf("enqueue prepare task: %w", err)
	}

	u.log.WithFields(logrus.Fields{
		"prepared_experiment_id": prep.ID,
		"experiment_id":          experimentID,
		"crop_alt":               cropAlt,
		"bgr_type":               bgrType,
	}).Info("prepare task enqueued via asynq")

	return prep, nil
}
