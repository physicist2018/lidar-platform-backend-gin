package usecase

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type GetAllUsersUseCase interface {
	Execute(ctx context.Context, filter *entity.UserFilter) (*pagination.Pagination[entity.User], error)
}

type GetUserByIDUseCase interface {
	Execute(ctx context.Context, id uint) (*entity.User, error)
}

type CreateUserUseCase interface {
	Execute(ctx context.Context, user *entity.User) error
}

type UpdateUserUseCase interface {
	Execute(ctx context.Context, user *entity.User) error
}

type DeleteUserUseCase interface {
	Execute(ctx context.Context, id uint) error
}
