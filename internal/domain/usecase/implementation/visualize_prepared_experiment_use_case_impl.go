package implementation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/queue"
)

type visualizePreparedExperimentUseCaseImpl struct {
	queueClient *queue.Client
	log         *logrus.Logger
}

var _ usecase.VisualizePreparedExperimentUseCase = (*visualizePreparedExperimentUseCaseImpl)(nil)

func NewVisualizePreparedExperimentUseCaseImpl(
	queueClient *queue.Client,
	log *logrus.Logger,
) *visualizePreparedExperimentUseCaseImpl {
	return &visualizePreparedExperimentUseCaseImpl{
		queueClient: queueClient,
		log:         log,
	}
}

func (u *visualizePreparedExperimentUseCaseImpl) Execute(
	ctx context.Context,
	prepID uint,
	wavelen float64,
	isPhoton int8,
	polarization string,
	vizType string,
	outputType string,
	formula string,
	regenerate bool,
	glued int8,
) (*usecase.AsyncTaskInfo, error) {
	payload := queue.VisualizePayload{
		PrepID:       prepID,
		Wavelen:      wavelen,
		IsPhoton:     isPhoton,
		Polarization: polarization,
		VizType:      vizType,
		OutputType:   outputType,
		Formula:      formula,
		Regenerate:   regenerate,
		Glued:        glued,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal visualize payload: %w", err)
	}

	task := asynq.NewTask(queue.TypeVisualize, data)
	info, err := u.queueClient.Enqueue(task)
	if err != nil {
		return nil, fmt.Errorf("enqueue visualize task: %w", err)
	}

	u.log.WithFields(logrus.Fields{
		"task_id": info.ID,
		"prep_id": prepID,
	}).Info("visualize task enqueued")

	return &usecase.AsyncTaskInfo{TaskID: info.ID}, nil
}
