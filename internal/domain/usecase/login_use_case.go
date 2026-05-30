package usecase

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/utils/auth"
)

type LoginUseCase interface {
	Execute(ctx context.Context, email, password string) (*auth.Claims, string, error)
}
