package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type LidarPackDataSource interface {
	SavePack(ctx context.Context, pack *entity.LidarPack) error
	// GetProfilesByExperimentID loads the main data LidarPack with all profiles for an experiment.
	GetProfilesByExperimentID(ctx context.Context, experimentID uint) ([]entity.LidarProfile, error)
	// GetProfilesByFileID loads all profiles for a specific file (used for BGR data).
	GetProfilesByFileID(ctx context.Context, fileID uint) ([]entity.LidarProfile, error)
}
