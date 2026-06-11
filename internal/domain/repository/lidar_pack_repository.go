package repository

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type LidarPackRepository interface {
	SavePack(ctx context.Context, pack *entity.LidarPack) error
}
