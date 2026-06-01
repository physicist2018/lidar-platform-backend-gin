package implementation

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/internal/utils/worker"
)

type createExperimentUseCaseImpl struct {
	repo       repository.ExperimentRepository
	minio      *storage.MinioClient
	workerPool *worker.Pool
	log        *logrus.Logger
}

var _ usecase.CreateExperimentUseCase = (*createExperimentUseCaseImpl)(nil)

func NewCreateExperimentUseCaseImpl(
	repo repository.ExperimentRepository,
	minio *storage.MinioClient,
	workerPool *worker.Pool,
	log *logrus.Logger,
) *createExperimentUseCaseImpl {
	return &createExperimentUseCaseImpl{
		repo:       repo,
		minio:      minio,
		workerPool: workerPool,
		log:        log,
	}
}

func (u *createExperimentUseCaseImpl) Execute(
	ctx context.Context,
	userID uint,
	title, comments string,
	licelZip, licelBgr, meteoFile *multipart.FileHeader,
) (*entity.Experiment, error) {
	// 1. Save files to temporary directory
	tempDir, err := os.MkdirTemp("", "experiment-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	tempZipPath := filepath.Join(tempDir, "licel.zip")
	tempBgrPath := filepath.Join(tempDir, "bgr.dat")
	tempMeteoPath := filepath.Join(tempDir, "meteo.dat")

	if err := saveUploadedFile(licelZip, tempZipPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("save licel zip: %w", err)
	}
	if err := saveUploadedFile(licelBgr, tempBgrPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("save bgr file: %w", err)
	}
	if err := saveUploadedFile(meteoFile, tempMeteoPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("save meteo file: %w", err)
	}

	// 2. Create Experiment with status "staged"
	exp := &entity.Experiment{
		UserID:   userID,
		Title:    title,
		Comments: comments,
		Status:   entity.StatusStaged,
	}

	if err := u.repo.Create(ctx, exp); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("create experiment: %w", err)
	}

	// 3. Submit preprocessing task to worker pool
	expID := exp.ID
	u.workerPool.Submit(func() {
		u.preprocess(expID, tempDir, tempZipPath, tempBgrPath, tempMeteoPath)
	})

	u.log.WithFields(logrus.Fields{
		"experiment_id": expID,
		"title":         title,
	}).Info("experiment created, preprocessing submitted to worker pool")

	return exp, nil
}

// preprocess runs in a background goroutine from the worker pool.
func (u *createExperimentUseCaseImpl) preprocess(expID uint, tempDir, zipPath, bgrPath, meteoPath string) {
	ctx := context.Background()
	log := u.log.WithField("experiment_id", expID)

	defer os.RemoveAll(tempDir)

	// 1. Update status → uploading
	if err := u.repo.Update(ctx, &entity.Experiment{
		ID:     expID,
		Status: entity.StatusUploading,
	}); err != nil {
		log.WithError(err).Error("failed to update status to uploading")
		u.setFailed(ctx, expID, err.Error())
		return
	}

	// 2. Parse licel zip to find MeasurementStartTime / MeasurementStopTime
	pack, err := licelformat.NewLicelPackFromZip(zipPath)
	if err != nil {
		log.WithError(err).Error("failed to parse licel zip")
		u.setFailed(ctx, expID, err.Error())
		return
	}
	if len(pack.Data) == 0 {
		errMsg := "licel zip contains no valid licel files"
		log.Error(errMsg)
		u.setFailed(ctx, expID, errMsg)
		return
	}

	var minStart, maxStop time.Time
	minStart = pack.StartTime
	maxStop = pack.StopTime

	// 3. Upload files to Minio
	basePath := fmt.Sprintf("experiments/%d/source", expID)
	zipObject := basePath + "/licel.zip"
	bgrObject := basePath + "/bgr.dat"
	meteoObject := basePath + "/meteo.dat"

	if err := u.minio.UploadFile(ctx, zipObject, zipPath, "application/zip"); err != nil {
		log.WithError(err).Error("failed to upload licel zip to minio")
		u.setFailed(ctx, expID, err.Error())
		return
	}
	if err := u.minio.UploadFile(ctx, bgrObject, bgrPath, "application/octet-stream"); err != nil {
		log.WithError(err).Error("failed to upload bgr file to minio")
		u.setFailed(ctx, expID, err.Error())
		return
	}
	if err := u.minio.UploadFile(ctx, meteoObject, meteoPath, "application/octet-stream"); err != nil {
		log.WithError(err).Error("failed to upload meteo file to minio")
		u.setFailed(ctx, expID, err.Error())
		return
	}

	// 4. Update experiment → done
	if err := u.repo.Update(ctx, &entity.Experiment{
		ID:                   expID,
		Status:               entity.StatusDone,
		MeasurementStartTime: &minStart,
		MeasurementStopTime:  &maxStop,
		LicelZipPath:         zipObject,
		LicelBgrPath:         bgrObject,
		MeteoFilePath:        meteoObject,
	}); err != nil {
		log.WithError(err).Error("failed to update experiment to done")
		u.setFailed(ctx, expID, err.Error())
		return
	}

	log.WithFields(logrus.Fields{
		"start_time": minStart,
		"stop_time":  maxStop,
		"files":      len(pack.Data),
	}).Info("experiment preprocessing completed successfully")
}

func (u *createExperimentUseCaseImpl) setFailed(ctx context.Context, expID uint, errMsg string) {
	if err := u.repo.Update(ctx, &entity.Experiment{
		ID:       expID,
		Status:   entity.StatusFailed,
		ErrorMsg: errMsg,
	}); err != nil {
		u.log.WithField("experiment_id", expID).WithError(err).
			Error("failed to set experiment status to failed")
	}
}

// saveUploadedFile saves a multipart file to disk.
func saveUploadedFile(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open uploaded file: %w", err)
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}

	return nil
}
