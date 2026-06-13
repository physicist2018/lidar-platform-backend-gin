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

type processExperimentUseCaseImpl struct {
	expRepo     repository.ExperimentRepository
	procRepo    repository.ProcessingRunRepository
	queueClient *queue.Client
	log         *logrus.Logger
}

var _ usecase.ProcessExperimentUseCase = (*processExperimentUseCaseImpl)(nil)

func NewProcessExperimentUseCaseImpl(
	expRepo repository.ExperimentRepository,
	procRepo repository.ProcessingRunRepository,
	queueClient *queue.Client,
	log *logrus.Logger,
) *processExperimentUseCaseImpl {
	return &processExperimentUseCaseImpl{
		expRepo:     expRepo,
		procRepo:    procRepo,
		queueClient: queueClient,
		log:         log,
	}
}

func (u *processExperimentUseCaseImpl) Execute(
	ctx context.Context,
	userID, experimentID uint,
	algorithm string,
	params json.RawMessage,
) (*entity.ProcessingRun, error) {
	// Validate that the experiment exists and is in "done" status
	exp, err := u.expRepo.FindByID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("experiment not found: %w", err)
	}

	if exp.Status != entity.StatusDone {
		return nil, &useCaseError{
			msg:    fmt.Sprintf("experiment %d must be in 'done' status, current: %s", experimentID, exp.Status),
			status: 409,
		}
	}

	// Remove all previous processing runs for this experiment,
	// including their dependents across all algorithms (cascade delete).
	// Each experiment can only have one active set of processing results.
	oldRuns, err := u.procRepo.FindByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("find old runs: %w", err)
	}
	if len(oldRuns) > 0 {
		ids := make([]uint, len(oldRuns))
		for i, r := range oldRuns {
			ids[i] = r.ID
		}
		u.log.WithFields(logrus.Fields{
			"experiment_id": experimentID,
			"algorithm":     algorithm,
			"old_count":     len(oldRuns),
		}).Info("removing old processing runs")

		if err := u.procRepo.DeleteCascade(ctx, ids); err != nil {
			return nil, fmt.Errorf("remove old runs: %w", err)
		}
	}

	run := &entity.ProcessingRun{
		UserID:       userID,
		ExperimentID: experimentID,
		Algorithm:    algorithm,
		Params:       params,
		Status:       entity.ProcStatusStaged,
	}

	if err := u.procRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create processing run: %w", err)
	}

	// Enqueue asynq task
	payload := queue.ProcessPayload{
		ProcID:       run.ID,
		ExperimentID: experimentID,
		Algorithm:    algorithm,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal process payload: %w", err)
	}

	task := asynq.NewTask(queue.TypeProcess, data)
	_, err = u.queueClient.Enqueue(task)
	if err != nil {
		// Mark as failed if enqueue fails
		_ = u.procRepo.Update(ctx, &entity.ProcessingRun{
			ID:       run.ID,
			Status:   entity.ProcStatusFailed,
			ErrorMsg: fmt.Sprintf("failed to enqueue task: %s", err.Error()),
		})
		return nil, fmt.Errorf("enqueue process task: %w", err)
	}

	u.log.WithFields(logrus.Fields{
		"processing_run_id": run.ID,
		"experiment_id":     experimentID,
		"algorithm":         algorithm,
	}).Info("process task enqueued via asynq")

	return run, nil
}

// useCaseError allows returning a specific HTTP status code from use cases.
type useCaseError struct {
	msg    string
	status int
}

func (e *useCaseError) Error() string   { return e.msg }
func (e *useCaseError) StatusCode() int { return e.status }
