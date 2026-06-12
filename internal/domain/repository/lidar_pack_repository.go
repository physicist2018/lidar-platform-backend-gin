package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type LidarPackRepository interface {
	SavePack(ctx context.Context, pack *entity.LidarPack) error
	// GetProfilesByExperimentID returns all profiles from the main data pack for an experiment.
	GetProfilesByExperimentID(ctx context.Context, experimentID uint) ([]entity.LidarProfile, error)
	// GetProfilesByFileID returns all profiles for a specific lidar file (e.g. BGR).
	GetProfilesByFileID(ctx context.Context, fileID uint) ([]entity.LidarProfile, error)
}
