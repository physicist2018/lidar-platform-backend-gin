package implementation

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/infrastructure/datasource/cache/key"
	"github.com/physicist2018/lidar-platform-go/internal/utils/pagination"
)

type UserCacheImpl struct {
	Redis *redis.Client
	TTL   time.Duration
	Log   *logrus.Logger
}

func NewUserCacheImpl(redis *redis.Client, ttl time.Duration, log *logrus.Logger) *UserCacheImpl {
	return &UserCacheImpl{Redis: redis, TTL: ttl, Log: log}
}

func (c *UserCacheImpl) buildFilterHash(filter *entity.UserFilter) string {
	raw := fmt.Sprintf("%d|%d|%s|%s|%s|%s",
		filter.Page, filter.Limit, filter.Sort, filter.Role, filter.Name, filter.Email)
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func (c *UserCacheImpl) GetAll(ctx context.Context, filter *entity.UserFilter) (*pagination.Pagination[entity.User], error) {
	cacheKey := key.UserListKey(c.buildFilterHash(filter))
	data, err := c.Redis.Get(ctx, cacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var result pagination.Pagination[entity.User]
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *UserCacheImpl) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	cacheKey := key.UserByIDKey(id)
	data, err := c.Redis.Get(ctx, cacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var user entity.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *UserCacheImpl) SetAll(ctx context.Context, filter *entity.UserFilter, data *pagination.Pagination[entity.User]) error {
	cacheKey := key.UserListKey(c.buildFilterHash(filter))
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.Redis.Set(ctx, cacheKey, payload, c.TTL).Err()
}

func (c *UserCacheImpl) SetByID(ctx context.Context, id uint, user *entity.User) error {
	cacheKey := key.UserByIDKey(id)
	payload, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return c.Redis.Set(ctx, cacheKey, payload, c.TTL).Err()
}

func (c *UserCacheImpl) Delete(ctx context.Context, id uint) error {
	return c.Redis.Del(ctx, key.UserByIDKey(id)).Err()
}

func (c *UserCacheImpl) DeleteAll(ctx context.Context) error {
	iter := c.Redis.Scan(ctx, 0, "user:*", 0).Iterator()
	for iter.Next(ctx) {
		if err := c.Redis.Del(ctx, iter.Val()).Err(); err != nil {
			c.Log.WithError(err).Warn("UserCache.DeleteAll: failed to delete key")
		}
	}
	return iter.Err()
}
