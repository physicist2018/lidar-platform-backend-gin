package repository

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	domainRepo "github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/datasource/persistance"
	"github.com/kshmirko/lidar-platform-go/internal/utils/response"
)

type LidarPackRepositoryImpl struct {
	DataSource persistance.LidarPackDataSource
	Log        *logrus.Logger
}

var _ domainRepo.LidarPackRepository = (*LidarPackRepositoryImpl)(nil)

func NewLidarPackRepositoryImpl(ds persistance.LidarPackDataSource, log *logrus.Logger) *LidarPackRepositoryImpl {
	return &LidarPackRepositoryImpl{DataSource: ds, Log: log}
}

func (r *LidarPackRepositoryImpl) SavePack(ctx context.Context, pack *entity.LidarPack) error {
	op := "LidarPackRepository.SavePack"
	if err := r.DataSource.SavePack(ctx, pack); err != nil {
		return response.InternalError(op, err)
	}
	return nil
}
