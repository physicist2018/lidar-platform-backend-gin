package usecase

import (
	"context"
	"encoding/json"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type ProcessExperimentUseCase interface {
	Execute(ctx context.Context, userID, experimentID uint, algorithm string, params json.RawMessage) (*entity.ProcessingRun, error)
}
