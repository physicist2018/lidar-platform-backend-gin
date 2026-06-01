package usecase

import (
	"context"
	"mime/multipart"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

type CreateExperimentUseCase interface {
	Execute(ctx context.Context, userID uint, title, comments string, licelZip, licelBgr, meteoFile *multipart.FileHeader) (*entity.Experiment, error)
}
