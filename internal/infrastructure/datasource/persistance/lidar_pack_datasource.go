package persistance

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type LidarPackDataSource interface {
	SavePack(ctx context.Context, pack *entity.LidarPack) error
}
