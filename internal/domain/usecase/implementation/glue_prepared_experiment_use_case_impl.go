package implementation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/internal/utils/worker"
)

type gluePreparedExperimentUseCaseImpl struct {
	prepRepo   repository.PreparedExperimentRepository
	minio      *storage.MinioClient
	workerPool *worker.Pool
	log        *logrus.Logger
}

var _ usecase.GluePreparedExperimentUseCase = (*gluePreparedExperimentUseCaseImpl)(nil)

func NewGluePreparedExperimentUseCaseImpl(
	prepRepo repository.PreparedExperimentRepository,
	minio *storage.MinioClient,
	workerPool *worker.Pool,
	log *logrus.Logger,
) *gluePreparedExperimentUseCaseImpl {
	return &gluePreparedExperimentUseCaseImpl{
		prepRepo:   prepRepo,
		minio:      minio,
		workerPool: workerPool,
		log:        log,
	}
}

func (u *gluePreparedExperimentUseCaseImpl) Execute(
	ctx context.Context,
	experimentID uint,
	wavelengths []float64,
	polarization string,
	h1, h2 float64,
) error {
	// Find PreparedExperiment by ExperimentID
	prep, err := u.prepRepo.FindByExperimentID(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("prepared experiment not found for experiment %d: %w", experimentID, err)
	}

	// Validate status
	if prep.Status != entity.PrepStatusDoneStageOne && prep.Status != entity.PrepStatusDoneStageTwo {
		return fmt.Errorf(
			"data must be in PrepStatusDoneStageOne or PrepStatusDoneStageTwo status, current: %s",
			prep.Status,
		)
	}

	prepID := prep.ID

	u.workerPool.Submit(func() {
		u.glueProcess(prepID, experimentID, prep.PathToData, wavelengths, polarization, h1, h2)
	})

	u.log.WithFields(logrus.Fields{
		"prepared_experiment_id": prepID,
		"experiment_id":          experimentID,
		"wavelengths":            wavelengths,
		"polarization":           polarization,
		"h1":                     h1,
		"h2":                     h2,
	}).Info("glue task submitted to worker pool")

	return nil
}

func (u *gluePreparedExperimentUseCaseImpl) glueProcess(
	prepID, experimentID uint,
	pathToData string,
	wavelengths []float64,
	polarization string,
	h1, h2 float64,
) {
	ctx := context.Background()
	log := u.log.WithField("prepared_experiment_id", prepID)

	tempDir, err := os.MkdirTemp("", "glue-*")
	if err != nil {
		log.WithError(err).Error("failed to create temp dir")
		u.setFailed(ctx, prepID, err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	// 1. Download prepared zip from MinIO
	localZipPath := filepath.Join(tempDir, "prepared.zip")
	if err := u.minio.DownloadFile(ctx, pathToData, localZipPath); err != nil {
		log.WithError(err).Error("failed to download prepared zip from minio")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 2. Parse the zip into LicelPack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		log.WithError(err).Error("failed to parse prepared zip")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 3. For each wavelength, perform glue
	for _, wvl := range wavelengths {
		if err := dataPack.Glue(wvl, h1, h2, polarization); err != nil {
			log.WithFields(logrus.Fields{
				"wavelength": wvl,
				"h1":         h1,
				"h2":         h2,
			}).WithError(err).Error("glue failed for wavelength")
			u.setFailed(ctx, prepID, fmt.Sprintf("glue failed for wavelength %.0f: %s", wvl, err.Error()))
			return
		}
	}

	// 4. Save updated pack back to zip
	outputZipPath := filepath.Join(tempDir, "glued.zip")
	if err := dataPack.SaveToZip(outputZipPath); err != nil {
		log.WithError(err).Error("failed to save glued zip")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 5. Upload glued zip to MinIO (overwrite)
	if err := u.minio.UploadFile(ctx, pathToData, outputZipPath, "application/zip"); err != nil {
		log.WithError(err).Error("failed to upload glued zip to minio")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 6. Update status to DoneStageTwo
	if err := u.prepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     prepID,
		Status: entity.PrepStatusDoneStageTwo,
	}); err != nil {
		log.WithError(err).Error("failed to update status to PrepStatusDoneStageTwo")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	log.WithFields(logrus.Fields{
		"experiment_id": experimentID,
		"wavelengths":   wavelengths,
		"h1":            h1,
		"h2":            h2,
	}).Info("glue completed successfully")
}

func (u *gluePreparedExperimentUseCaseImpl) setFailed(ctx context.Context, prepID uint, errMsg string) {
	if err := u.prepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:       prepID,
		Status:   entity.PrepStatusFailed,
		ErrorMsg: errMsg,
	}); err != nil {
		u.log.WithField("prepared_experiment_id", prepID).WithError(err).
			Error("failed to set prepared experiment status to failed")
	}
}
