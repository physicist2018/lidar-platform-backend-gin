package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type ProcessedSignalDataSource interface {
	BatchCreate(ctx context.Context, signals []entity.ProcessedSignal) error
	GetByProcessingRunID(ctx context.Context, runID uint) ([]entity.ProcessedSignal, error)
	GetByProcessingRunIDFiltered(ctx context.Context, runID uint, filter entity.ProcessedSignalFilter) ([]entity.ProcessedSignal, error)
	DeleteByProcessingRunIDs(ctx context.Context, runIDs []uint) error
}
