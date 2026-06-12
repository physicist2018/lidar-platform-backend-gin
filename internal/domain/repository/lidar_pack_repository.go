package repository

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type LidarPackRepository interface {
	SavePack(ctx context.Context, pack *entity.LidarPack) error
}
