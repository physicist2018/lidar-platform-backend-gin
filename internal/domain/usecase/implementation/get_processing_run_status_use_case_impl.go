package implementation

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
)

type getProcessingRunStatusUseCaseImpl struct {
	procRepo repository.ProcessingRunRepository
	log      *logrus.Logger
}

var _ usecase.GetProcessingRunStatusUseCase = (*getProcessingRunStatusUseCaseImpl)(nil)

func NewGetProcessingRunStatusUseCaseImpl(
	procRepo repository.ProcessingRunRepository,
	log *logrus.Logger,
) *getProcessingRunStatusUseCaseImpl {
	return &getProcessingRunStatusUseCaseImpl{
		procRepo: procRepo,
		log:      log,
	}
}

func (u *getProcessingRunStatusUseCaseImpl) Execute(ctx context.Context, id uint) (*entity.ProcessingRun, error) {
	run, err := u.procRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return run, nil
}
