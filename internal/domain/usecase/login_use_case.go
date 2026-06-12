package usecase

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/utils/auth"
)

type LoginUseCase interface {
	Execute(ctx context.Context, email, password string) (*auth.Claims, string, error)
}
