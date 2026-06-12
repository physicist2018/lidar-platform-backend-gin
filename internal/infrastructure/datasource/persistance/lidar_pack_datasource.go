package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type LidarPackDataSource interface {
	SavePack(ctx context.Context, pack *entity.LidarPack) error
}
