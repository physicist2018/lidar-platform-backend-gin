package implementation

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/internal/utils/worker"
)

type prepareExperimentUseCaseImpl struct {
	expRepo    repository.ExperimentRepository
	prepRepo   repository.PreparedExperimentRepository
	minio      *storage.MinioClient
	workerPool *worker.Pool
	log        *logrus.Logger
}

var _ usecase.PrepareExperimentUseCase = (*prepareExperimentUseCaseImpl)(nil)

func NewPrepareExperimentUseCaseImpl(
	expRepo repository.ExperimentRepository,
	prepRepo repository.PreparedExperimentRepository,
	minio *storage.MinioClient,
	workerPool *worker.Pool,
	log *logrus.Logger,
) *prepareExperimentUseCaseImpl {
	return &prepareExperimentUseCaseImpl{
		expRepo:    expRepo,
		prepRepo:   prepRepo,
		minio:      minio,
		workerPool: workerPool,
		log:        log,
	}
}

func (u *prepareExperimentUseCaseImpl) Execute(
	ctx context.Context,
	experimentID uint,
	cropAlt float64,
	bgrType entity.BGRType,
	bgrAlt float64,
) (*entity.PreparedExperiment, error) {
	// Validate that the experiment exists
	exp, err := u.expRepo.FindByID(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("experiment not found: %w", err)
	}

	if exp.Status != entity.StatusDone {
		return nil, fmt.Errorf("experiment %d must be in 'done' status, current: %s", experimentID, exp.Status)
	}

	// Build the target Minio path for processed data
	pathToData := fmt.Sprintf("experiments/%d/processed/dats.zip", experimentID)

	prep := &entity.PreparedExperiment{
		ExperimentID: experimentID,
		CropAlt:      cropAlt,
		BGRType:      bgrType,
		BGRAlt:       bgrAlt,
		PathToData:   pathToData,
		Status:       entity.PrepStatusStaged,
	}

	if err := prep.ValidateBGRParams(); err != nil {
		return nil, err
	}

	if err := u.prepRepo.Create(ctx, prep); err != nil {
		return nil, fmt.Errorf("create prepared experiment: %w", err)
	}

	prepID := prep.ID
	u.workerPool.Submit(func() {
		u.preprocess(prepID, experimentID, cropAlt, bgrType, bgrAlt, pathToData)
	})

	u.log.WithFields(logrus.Fields{
		"prepared_experiment_id": prepID,
		"experiment_id":          experimentID,
		"crop_alt":               cropAlt,
		"bgr_type":               bgrType,
	}).Info("prepared experiment created, preprocessing submitted to worker pool")

	return prep, nil
}

func (u *prepareExperimentUseCaseImpl) preprocess(
	prepID, experimentID uint,
	cropAlt float64,
	bgrType entity.BGRType,
	bgrAlt float64,
	pathToData string,
) {
	ctx := context.Background()
	log := u.log.WithField("prepared_experiment_id", prepID)

	tempDir, err := os.MkdirTemp("", "prepare-*")
	if err != nil {
		log.WithError(err).Error("failed to create temp dir")
		u.setFailed(ctx, prepID, err.Error())
		return
	}
	defer os.RemoveAll(tempDir)

	// 1. Download source data zip from Minio
	srcZipObject := fmt.Sprintf("experiments/%d/source/licel.zip", experimentID)
	localZipPath := filepath.Join(tempDir, "data.zip")

	if err := u.minio.DownloadFile(ctx, srcZipObject, localZipPath); err != nil {
		log.WithError(err).Error("failed to download data zip from minio")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 2. Parse data pack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		log.WithError(err).Error("failed to parse data zip")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 3. Status → removebgr
	if err := u.prepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     prepID,
		Status: entity.PrepStatusRemoveBGR,
	}); err != nil {
		log.WithError(err).Error("failed to update status to removebgr")
	}

	// 4. Background subtraction
	if err := u.removeBackground(dataPack, bgrType, bgrAlt, tempDir, experimentID); err != nil {
		log.WithError(err).Error("background subtraction failed")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 5. Status → cropping
	if err := u.prepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     prepID,
		Status: entity.PrepStatusCropping,
	}); err != nil {
		log.WithError(err).Error("failed to update status to cropping")
	}

	// 6. Crop by altitude
	if err := dataPack.SetMaxDist(cropAlt); err != nil {
		log.WithError(err).Error("cropping failed")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 7. Save processed data to zip
	processedZipPath := filepath.Join(tempDir, "dats.zip")
	if err := dataPack.SaveToZip(processedZipPath); err != nil {
		log.WithError(err).Error("failed to save processed zip")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 8. Upload processed zip to Minio
	if err := u.minio.UploadFile(ctx, pathToData, processedZipPath, "application/zip"); err != nil {
		log.WithError(err).Error("failed to upload processed zip to minio")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	// 9. Mark as done
	if err := u.prepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     prepID,
		Status: entity.PrepStatusDone,
	}); err != nil {
		log.WithError(err).Error("failed to update status to done")
		u.setFailed(ctx, prepID, err.Error())
		return
	}

	log.WithFields(logrus.Fields{
		"experiment_id":  experimentID,
		"processed_path": pathToData,
		"crop_alt":       cropAlt,
		"bgr_type":       bgrType,
	}).Info("experiment preparation completed successfully")
}

func (u *prepareExperimentUseCaseImpl) removeBackground(
	dataPack *licelformat.LicelPack,
	bgrType entity.BGRType,
	bgrAlt float64,
	tempDir string,
	experimentID uint,
) error {
	switch bgrType {
	case entity.BGRFile:
		return u.removeBGRFile(dataPack, tempDir, experimentID)
	case entity.BGRAvgTail:
		return u.removeBGRTailStat(dataPack, bgrAlt, avg)
	case entity.BGRMedTail:
		return u.removeBGRTailStat(dataPack, bgrAlt, median)
	default:
		return fmt.Errorf("unknown bgr_type: %s", bgrType)
	}
}

// removeBGRFile downloads the background file from Minio and subtracts matching profiles.
func (u *prepareExperimentUseCaseImpl) removeBGRFile(
	dataPack *licelformat.LicelPack,
	tempDir string,
	experimentID uint,
) error {
	ctx := context.Background()

	bgrObj := fmt.Sprintf("experiments/%d/source/bgr.dat", experimentID)
	localBgrPath := filepath.Join(tempDir, "bgr.dat")

	if err := u.minio.DownloadFile(ctx, bgrObj, localBgrPath); err != nil {
		return fmt.Errorf("download bgr file: %w", err)
	}

	bgrFile, err := licelformat.LoadLicelFile(localBgrPath)
	if err != nil {
		return fmt.Errorf("parse bgr file: %w", err)
	}

	// Subtract matching profiles
	for fname, dataFile := range dataPack.Data {
		for i := range dataFile.Profiles {
			sigProf := &dataFile.Profiles[i]

			bgrProf, found := bgrFile.SelectProfile(
				sigProf.Photon,
				sigProf.Wavelength,
				sigProf.Polarization,
			)
			if !found {
				u.log.WithFields(logrus.Fields{
					"file":         fname,
					"photon":       sigProf.Photon,
					"wavelength":   sigProf.Wavelength,
					"polarization": sigProf.Polarization,
				}).Warn("no matching bgr profile found, skipping subtraction")
				continue
			}

			minLen := len(sigProf.Data)
			if len(bgrProf.Data) < minLen {
				minLen = len(bgrProf.Data)
			}
			for j := 0; j < minLen; j++ {
				sigProf.Data[j] -= bgrProf.Data[j]
			}
		}
		dataPack.Data[fname] = dataFile
	}

	return nil
}

// tailStat is a function type for statistical operations.
type tailStat func(data []float64) float64

func avg(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func median(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2.0
	}
	return sorted[mid]
}

// removeBGRTailStat subtracts the avg or median of tail values from each profile.
func (u *prepareExperimentUseCaseImpl) removeBGRTailStat(
	dataPack *licelformat.LicelPack,
	bgrAlt float64,
	statFn tailStat,
) error {
	for fname, dataFile := range dataPack.Data {
		for i := range dataFile.Profiles {
			prof := &dataFile.Profiles[i]
			tailValues := prof.Data

			// Determine the first index above bgrAlt
			if prof.BinWidth > 0 {
				startIdx := int(math.Ceil(bgrAlt / prof.BinWidth))
				if startIdx < 0 {
					startIdx = 0
				}
				if startIdx < len(prof.Data) {
					tailValues = prof.Data[startIdx:]
				}
			}

			bgValue := statFn(tailValues)
			for j := range prof.Data {
				prof.Data[j] -= bgValue
			}
		}
		dataPack.Data[fname] = dataFile
	}

	return nil
}

func (u *prepareExperimentUseCaseImpl) setFailed(ctx context.Context, prepID uint, errMsg string) {
	if err := u.prepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:       prepID,
		Status:   entity.PrepStatusFailed,
		ErrorMsg: errMsg,
	}); err != nil {
		u.log.WithField("prepared_experiment_id", prepID).WithError(err).
			Error("failed to set prepared experiment status to failed")
	}
}
