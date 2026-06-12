package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/cache"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
	"github.com/physicist2018/lidar-platform-go/internal/utils/response"
)

type UserRepositoryImpl struct {
	DataSource persistance.UserDataSource
	Cache      cache.UserCache
	Log        *logrus.Logger
}

var _ domainRepo.UserRepository = (*UserRepositoryImpl)(nil)

func NewUserRepositoryImpl(ds persistance.UserDataSource, c cache.UserCache, log *logrus.Logger) *UserRepositoryImpl {
	return &UserRepositoryImpl{DataSource: ds, Cache: c, Log: log}
}

func (r *UserRepositoryImpl) FindAll(ctx context.Context, filter *entity.UserFilter) (*pagination.Pagination[entity.User], error) {
	op := "UserRepository.FindAll"

	if cached, err := r.Cache.GetAll(ctx, filter); err == nil {
		return cached, nil
	}

	users, total, err := r.DataSource.GetAll(ctx, filter)
	if err != nil {
		return nil, response.InternalError(op, err)
	}

	result := pagination.New(users, total, filter.Page, filter.Limit)

	_ = r.Cache.SetAll(ctx, filter, result)

	return result, nil
}

func (r *UserRepositoryImpl) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	op := "UserRepository.FindByID"

	if cached, err := r.Cache.GetByID(ctx, id); err == nil {
		return cached, nil
	}

	user, err := r.DataSource.GetByID(ctx, id)
	if err != nil {
		return nil, response.InternalError(op, err)
	}

	_ = r.Cache.SetByID(ctx, id, user)

	return user, nil
}

func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return r.DataSource.GetByEmail(ctx, email)
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *entity.User) error {
	op := "UserRepository.Create"

	if err := r.DataSource.Create(ctx, user); err != nil {
		return response.InternalError(op, err)
	}

	_ = r.Cache.DeleteAll(ctx)

	return nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *entity.User) error {
	op := "UserRepository.Update"

	if err := r.DataSource.Update(ctx, user); err != nil {
		return response.InternalError(op, err)
	}

	_ = r.Cache.Delete(ctx, user.ID)

	return nil
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, id uint) error {
	op := "UserRepository.Delete"

	if err := r.DataSource.Delete(ctx, id); err != nil {
		return response.InternalError(op, err)
	}

	_ = r.Cache.Delete(ctx, id)

	return nil
}
