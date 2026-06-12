package persistance

import (
	"context"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
)

type UserDataSource interface {
	GetAll(ctx context.Context, filter *entity.UserFilter) ([]entity.User, int64, error)
	GetByID(ctx context.Context, id uint) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
