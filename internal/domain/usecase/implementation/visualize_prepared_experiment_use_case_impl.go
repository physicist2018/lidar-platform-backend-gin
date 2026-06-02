package implementation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/pkg/visualize"
)

const (
	defaultOutputType = "svg"
	presignExpiry     = 1 * time.Hour
)

type visualizePreparedExperimentUseCaseImpl struct {
	prepRepo   repository.PreparedExperimentRepository
	chartRepo  repository.ExperimentChartRepository
	minio      *storage.MinioClient
	log        *logrus.Logger
}

var _ usecase.VisualizePreparedExperimentUseCase = (*visualizePreparedExperimentUseCaseImpl)(nil)

func NewVisualizePreparedExperimentUseCaseImpl(
	prepRepo repository.PreparedExperimentRepository,
	chartRepo repository.ExperimentChartRepository,
	minio *storage.MinioClient,
	log *logrus.Logger,
) *visualizePreparedExperimentUseCaseImpl {
	return &visualizePreparedExperimentUseCaseImpl{
		prepRepo:  prepRepo,
		chartRepo: chartRepo,
		minio:     minio,
		log:       log,
	}
}

// namedProfile holds a profile together with its source file metadata.
type namedProfile struct {
	LicelProfile licelformat.LicelProfile
	StartTime    float64 // unix seconds
	Filename     string
}

func (u *visualizePreparedExperimentUseCaseImpl) Execute(
	ctx context.Context,
	prepID uint,
	wavelen float64,
	isPhoton int8,
	polarization string,
	vizType string,
	outputType string,
	formula string,
	regenerate bool,
) (string, error) {
	if outputType == "" {
		outputType = defaultOutputType
	}
	if formula == "" {
		formula = "raw"
	}
	if formula != "raw" && formula != "rangecorr" && formula != "lograngecorr" {
		return "", fmt.Errorf("unknown formula: %s (valid: raw, rangecorr, lograngecorr)", formula)
	}

	// 1. Find PreparedExperiment to get ExperimentID
	prep, err := u.prepRepo.FindByID(ctx, prepID)
	if err != nil {
		return "", fmt.Errorf("prepared experiment not found: %w", err)
	}
	if prep.Status != entity.PrepStatusDone {
		return "", fmt.Errorf("prepared experiment %d is not ready (status: %s)", prep.ID, prep.Status)
	}

	experimentID := prep.ExperimentID

	// 2. If not forced regenerate, try to find cached chart in DB
	if !regenerate {
		cached, err := u.chartRepo.FindByParams(ctx, experimentID, vizType, formula, wavelen, polarization, isPhoton)
		if err != nil {
			u.log.WithError(err).Warn("failed to lookup cached chart, will regenerate")
		}
		if cached != nil {
			u.log.WithField("path", cached.PathToObject).Info("returning cached chart")
			return u.minio.PresignedGetObject(ctx, cached.PathToObject, presignExpiry)
		}
	}

	// 3. Download prepared zip from Minio
	tempDir, err := os.MkdirTemp("", "visualize-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	localZipPath := filepath.Join(tempDir, "prepared.zip")
	if err := u.minio.DownloadFile(ctx, prep.PathToData, localZipPath); err != nil {
		return "", fmt.Errorf("download prepared data: %w", err)
	}

	// 4. Parse the zip into LicelPack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		return "", fmt.Errorf("parse prepared zip: %w", err)
	}

	// 5. Extract matching profiles with time metadata
	profiles := u.extractProfiles(dataPack, isPhoton != 0, wavelen, polarization)
	if len(profiles) == 0 {
		return "", fmt.Errorf(
			"no profiles found for wavelen=%.0f isPhoton=%d polarization=%s",
			wavelen, isPhoton, polarization,
		)
	}

	// Sort by start time
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].StartTime < profiles[j].StartTime
	})

	// 6. Generate visualization
	var result *usecase.VisualizeResult
	switch vizType {
	case "image":
		result, err = u.genHeatmap(profiles, outputType, formula)
	case "profile":
		result, err = u.genProfile(profiles, outputType, formula)
	default:
		return "", fmt.Errorf("unknown visualization type: %s", vizType)
	}
	if err != nil {
		return "", err
	}

	// 7. Upload to Minio and save record to DB
	ext := outputType
	if ext == "svg" {
		ext = "svg"
	} else if ext == "json" {
		ext = "json"
	} else if ext == "png" {
		ext = "png"
	}

	objectPath := fmt.Sprintf("experiments/%d/images/%s-%.0f-%s-%d-%s.%s",
		experimentID, vizType, wavelen, polarization, isPhoton, formula, ext)

	if err := u.minio.UploadBytes(ctx, objectPath, result.Body, result.ContentType); err != nil {
		return "", fmt.Errorf("upload chart to minio: %w", err)
	}

	chart := &entity.ExperimentChart{
		ExperimentID: experimentID,
		ChartType:    vizType,
		Formula:      formula,
		Wavelen:      wavelen,
		Polarization: polarization,
		IsPhoton:     isPhoton,
		PathToObject: objectPath,
	}
	if err := u.chartRepo.Create(ctx, chart); err != nil {
		u.log.WithError(err).Warn("failed to save experiment chart record (chart is still uploaded)")
		// Non-fatal: the chart was uploaded, just could not save DB record
	}

	return u.minio.PresignedGetObject(ctx, objectPath, presignExpiry)
}

// extractProfiles walks all files in the pack and collects matching profiles.
func (u *visualizePreparedExperimentUseCaseImpl) extractProfiles(
	dataPack *licelformat.LicelPack,
	isPhoton bool,
	wavelen float64,
	polarization string,
) []namedProfile {
	var result []namedProfile
	for fname, licf := range dataPack.Data {
		prof, found := licf.SelectProfile(isPhoton, wavelen, polarization)
		if !found {
			continue
		}
		result = append(result, namedProfile{
			LicelProfile: prof,
			StartTime:    float64(licf.MeasurementStartTime.Unix()),
			Filename:     fname,
		})
	}
	return result
}

func (u *visualizePreparedExperimentUseCaseImpl) genHeatmap(
	profiles []namedProfile,
	outputType string,
	formula string,
) (*usecase.VisualizeResult, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles for heatmap")
	}

	maxBins := 0
	for _, p := range profiles {
		if len(p.LicelProfile.Data) > maxBins {
			maxBins = len(p.LicelProfile.Data)
		}
	}

	if maxBins == 0 {
		return nil, fmt.Errorf("profile data is empty")
	}

	// Use the first profile's bin width for distance calculation
	binWidth := profiles[0].LicelProfile.BinWidth
	if binWidth <= 0 {
		binWidth = 7.5
	}

	// Time axis: Unix seconds + HH:MM labels
	nTime := len(profiles)
	timeLabels := make([]string, nTime)
	for i, p := range profiles {
		timeLabels[i] = visualize.FormatTimeHHMM(int64(p.StartTime))
	}

	// Distance axis
	distanceLabels := make([]string, maxBins)
	for i := 0; i < maxBins; i++ {
		distanceLabels[i] = fmt.Sprintf("%.0f", float64(i)*binWidth)
	}

	// zData[time][distance] with formula applied
	var titleSuffix string
	zData := make([][]float64, nTime)
	for i, p := range profiles {
		row := make([]float64, maxBins)
		copy(row, p.LicelProfile.Data)
		visualize.ApplyFormula(row, formula, binWidth)
		zData[i] = row
	}
	switch formula {
	case "rangecorr":
		titleSuffix = " (P × r²)"
	case "lograngecorr":
		titleSuffix = " (ℓоg₁₀(P × r²))"
	default:
		titleSuffix = " (raw signal)"
	}

	switch outputType {
	case "json":
		r, err := visualize.HeatmapToPlotly(timeLabels, distanceLabels, zData, titleSuffix)
		return toUseCaseResult(r), err
	case "png":
		r, err := visualize.HeatmapToPNG(timeLabels, distanceLabels, zData, titleSuffix)
		return toUseCaseResult(r), err
	default:
		r, err := visualize.HeatmapToSVG(timeLabels, distanceLabels, zData, titleSuffix)
		return toUseCaseResult(r), err
	}
}

func (u *visualizePreparedExperimentUseCaseImpl) genProfile(
	profiles []namedProfile,
	outputType string,
	formula string,
) (*usecase.VisualizeResult, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles to average")
	}

	binWidth := profiles[0].LicelProfile.BinWidth
	if binWidth <= 0 {
		binWidth = 7.5
	}

	// Determine the max length among all profiles
	maxLen := 0
	for _, p := range profiles {
		if len(p.LicelProfile.Data) > maxLen {
			maxLen = len(p.LicelProfile.Data)
		}
	}

	// Average element-wise
	avgData := make([]float64, maxLen)
	counts := make([]int, maxLen)
	for _, p := range profiles {
		for j, v := range p.LicelProfile.Data {
			avgData[j] += v
			counts[j]++
		}
	}
	for j := 0; j < maxLen; j++ {
		if counts[j] > 0 {
			avgData[j] /= float64(counts[j])
		}
	}

	// Apply formula to averaged data
	visualize.ApplyFormula(avgData, formula, binWidth)

	var titleSuffix string
	switch formula {
	case "rangecorr":
		titleSuffix = " (P × r²)"
	case "lograngecorr":
		titleSuffix = " (ℓог₁₀(P × r²))"
	default:
		titleSuffix = " (raw signal)"
	}

	// Build distance axis
	distance := make([]float64, maxLen)
	for i := 0; i < maxLen; i++ {
		distance[i] = float64(i) * binWidth
	}

	switch outputType {
	case "json":
		r, err := visualize.ProfileToPlotly(distance, avgData, titleSuffix)
		return toUseCaseResult(r), err
	case "png":
		r, err := visualize.ProfileToPNG(distance, avgData, titleSuffix)
		return toUseCaseResult(r), err
	default:
		r, err := visualize.ProfileToSVG(distance, avgData, titleSuffix)
		return toUseCaseResult(r), err
	}
}

// toUseCaseResult converts a pkg/visualize.Result into a usecase.VisualizeResult.
func toUseCaseResult(r *visualize.Result) *usecase.VisualizeResult {
	if r == nil {
		return nil
	}
	return &usecase.VisualizeResult{
		ContentType: r.ContentType,
		Body:        r.Body,
	}
}
