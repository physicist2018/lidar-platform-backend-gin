package implementation

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
	"github.com/physicist2018/lidar-platform-go/internal/domain/usecase"
)

type getStage0DataUseCaseImpl struct {
	procRunRepo repository.ProcessingRunRepository
	procSigRepo repository.ProcessedSignalRepository
	expRepo     repository.ExperimentRepository
	log         *logrus.Logger
}

var _ usecase.GetStage0DataUseCase = (*getStage0DataUseCaseImpl)(nil)

func NewGetStage0DataUseCaseImpl(
	procRunRepo repository.ProcessingRunRepository,
	procSigRepo repository.ProcessedSignalRepository,
	expRepo repository.ExperimentRepository,
	log *logrus.Logger,
) *getStage0DataUseCaseImpl {
	return &getStage0DataUseCaseImpl{
		procRunRepo: procRunRepo,
		procSigRepo: procSigRepo,
		expRepo:     expRepo,
		log:         log,
	}
}

func (u *getStage0DataUseCaseImpl) Execute(
	ctx context.Context,
	runID uint,
	wavelength float64,
	polarization string,
	deviceID string,
	timeFrom *time.Time,
	timeTo *time.Time,
) (*entity.Stage0DataResult, error) {
	// 1. Load the processing run
	run, err := u.procRunRepo.FindByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load processing run: %w", err)
	}
	if run.Algorithm != "stage0" {
		return nil, fmt.Errorf("run %d is not a stage0 run (algorithm=%s)", runID, run.Algorithm)
	}
	if run.Status != entity.ProcStatusDone {
		return nil, fmt.Errorf("run %d is not done (status=%s)", runID, run.Status)
	}

	// 2. Load the experiment for time defaults
	exp, err := u.expRepo.FindByID(ctx, run.ExperimentID)
	if err != nil {
		return nil, fmt.Errorf("load experiment: %w", err)
	}

	// 3. Apply defaults for time bounds
	if timeFrom == nil {
		timeFrom = exp.MeasurementStartTime
	}
	if timeTo == nil {
		timeTo = exp.MeasurementStopTime
	}

	// 4. Build the filter
	filter := entity.ProcessedSignalFilter{
		Wavelength:   &wavelength,
		Polarization: &polarization,
		DeviceID:     &deviceID,
		TimeFrom:     timeFrom,
		TimeTo:       timeTo,
	}

	// 5. Query processed signals
	signals, err := u.procSigRepo.FindByProcessingRunIDFiltered(ctx, runID, filter)
	if err != nil {
		return nil, fmt.Errorf("query processed signals: %w", err)
	}

	if len(signals) == 0 {
		return &entity.Stage0DataResult{
			Distance: []float64{},
			Data:     [][]float64{},
			Time:     []time.Time{},
		}, nil
	}

	// 6. Build distance axis from first profile's BinWidth
	binWidth := signals[0].BinWidth
	signalLen := len(signals[0].Signal)
	distance := make([]float64, signalLen)
	for i := 0; i < signalLen; i++ {
		distance[i] = float64(i) * binWidth
	}

	// 7. Build data and time slices
	data := make([][]float64, len(signals))
	times := make([]time.Time, len(signals))
	for i, sig := range signals {
		data[i] = sig.Signal
		if sig.FileStartTime != nil {
			times[i] = *sig.FileStartTime
		}
	}

	// Round distance to a sensible precision (cm)
	for i := range distance {
		distance[i] = math.Round(distance[i]*100) / 100
	}

	return &entity.Stage0DataResult{
		Distance: distance,
		Data:     data,
		Time:     times,
	}, nil
}
