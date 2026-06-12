package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type ProcessedSignalRepository interface {
	BatchCreate(ctx context.Context, signals []entity.ProcessedSignal) error
	FindByProcessingRunID(ctx context.Context, runID uint) ([]entity.ProcessedSignal, error)
}
