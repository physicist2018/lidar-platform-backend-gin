package repository

import (
	"context"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/utils/pagination"
)

type UserRepository interface {
	FindAll(ctx context.Context, filter *entity.UserFilter) (*pagination.Pagination[entity.User], error)
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
