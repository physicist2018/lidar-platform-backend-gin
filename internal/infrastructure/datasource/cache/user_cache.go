package cache

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type UserCache interface {
	GetAll(ctx context.Context, filter *entity.UserFilter) (*pagination.Pagination[entity.User], error)
	GetByID(ctx context.Context, id uint) (*entity.User, error)
	SetAll(ctx context.Context, filter *entity.UserFilter, data *pagination.Pagination[entity.User]) error
	SetByID(ctx context.Context, id uint, user *entity.User) error
	Delete(ctx context.Context, id uint) error
	DeleteAll(ctx context.Context) error
}
