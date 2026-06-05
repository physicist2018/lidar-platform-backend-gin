package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hibiken/asynq"
	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
	"github.com/kshmirko/lidar-platform-go/pkg/visualize"
)

const (
	defaultOutputType = "png"
	presignExpiry     = 1 * time.Hour
)

// HandlerDeps holds all dependencies needed by the async task handlers.
type HandlerDeps struct {
	PrepRepo  repository.PreparedExperimentRepository
	ChartRepo repository.ExperimentChartRepository
	Minio     *storage.MinioClient
	TaskStore *TaskStore
	Log       *logrus.Logger
}

// NewServeMux registers all task handlers and returns the mux.
func NewServeMux(deps *HandlerDeps) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TypePrepare, deps.handlePrepare)
	mux.HandleFunc(TypeGlue, deps.handleGlue)
	mux.HandleFunc(TypeVisualize, deps.handleVisualize)
	return mux
}

// ---------- Prepare handler ----------

func (d *HandlerDeps) handlePrepare(ctx context.Context, t *asynq.Task) error {
	var payload PreparePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal prepare payload: %w", err)
	}

	log := d.Log.WithFields(logrus.Fields{
		"task_id":       t.ResultWriter().TaskID(),
		"prep_id":       payload.PrepID,
		"experiment_id": payload.ExperimentID,
	})

	tempDir, err := os.MkdirTemp("", "prepare-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Download source data zip from Minio
	srcZipObject := fmt.Sprintf("experiments/%d/source/licel.zip", payload.ExperimentID)
	localZipPath := filepath.Join(tempDir, "data.zip")
	if err := d.Minio.DownloadFile(ctx, srcZipObject, localZipPath); err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("download data zip: %s", err.Error()))
	}

	// 2. Parse data pack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("parse data zip: %s", err.Error()))
	}

	// 3. Status → removebgr
	_ = d.PrepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     payload.PrepID,
		Status: entity.PrepStatusRemoveBGR,
	})

	// 4. Background subtraction
	bgrType := entity.BGRType(payload.BGRType)
	if err := d.removeBackground(dataPack, bgrType, payload.BGRAlt, tempDir, payload.ExperimentID); err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("background subtraction: %s", err.Error()))
	}

	// 5. Status → cropping
	_ = d.PrepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     payload.PrepID,
		Status: entity.PrepStatusCropping,
	})

	// 6. Crop by altitude
	if err := dataPack.SetMaxDist(payload.CropAlt); err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("cropping: %s", err.Error()))
	}

	// 7. Save processed data to zip
	processedZipPath := filepath.Join(tempDir, "dats.zip")
	if err := dataPack.SaveToZip(processedZipPath); err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("save processed zip: %s", err.Error()))
	}

	// 8. Upload processed zip to Minio
	if err := d.Minio.UploadFile(ctx, payload.PathToData, processedZipPath, "application/zip"); err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("upload processed zip: %s", err.Error()))
	}

	// 9. Mark as done stage one
	if err := d.PrepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     payload.PrepID,
		Status: entity.PrepStatusDoneStageOne,
	}); err != nil {
		return d.setPrepFailed(ctx, payload.PrepID, fmt.Sprintf("update status: %s", err.Error()))
	}

	log.WithFields(logrus.Fields{
		"processed_path": payload.PathToData,
		"crop_alt":       payload.CropAlt,
	}).Info("preparation completed successfully")

	return nil
}

func (d *HandlerDeps) setPrepFailed(ctx context.Context, prepID uint, errMsg string) error {
	log := d.Log.WithField("prep_id", prepID).WithError(fmt.Errorf("%s", errMsg))
	log.Error("prepare task failed")
	if err := d.PrepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:       prepID,
		Status:   entity.PrepStatusFailed,
		ErrorMsg: errMsg,
	}); err != nil {
		log.WithError(err).Error("failed to set prepared experiment status to failed")
	}
	return fmt.Errorf("prepare failed for prep %d: %s", prepID, errMsg)
}

func (d *HandlerDeps) removeBackground(
	dataPack *licelformat.LicelPack,
	bgrType entity.BGRType,
	bgrAlt float64,
	tempDir string,
	experimentID uint,
) error {
	switch bgrType {
	case entity.BGRFile:
		return d.removeBGRFile(dataPack, tempDir, experimentID)
	case entity.BGRAvgTail:
		return removeBGRTailStat(dataPack, bgrAlt, avg)
	case entity.BGRMedTail:
		return removeBGRTailStat(dataPack, bgrAlt, median)
	default:
		return fmt.Errorf("unknown bgr_type: %s", bgrType)
	}
}

func (d *HandlerDeps) removeBGRFile(
	dataPack *licelformat.LicelPack,
	tempDir string,
	experimentID uint,
) error {
	ctx := context.Background()

	bgrObj := fmt.Sprintf("experiments/%d/source/bgr.dat", experimentID)
	localBgrPath := filepath.Join(tempDir, "bgr.dat")

	if err := d.Minio.DownloadFile(ctx, bgrObj, localBgrPath); err != nil {
		return fmt.Errorf("download bgr file: %w", err)
	}

	bgrFile, err := licelformat.LoadLicelFile(localBgrPath)
	if err != nil {
		return fmt.Errorf("parse bgr file: %w", err)
	}

	for fname, dataFile := range dataPack.Data {
		for i := range dataFile.Profiles {
			sigProf := &dataFile.Profiles[i]

			bgrProf, found := bgrFile.SelectProfile(
				sigProf.Photon,
				sigProf.Wavelength,
				sigProf.Polarization,
			)
			if !found {
				d.Log.WithFields(logrus.Fields{
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

func removeBGRTailStat(
	dataPack *licelformat.LicelPack,
	bgrAlt float64,
	statFn func([]float64) float64,
) error {
	for fname, dataFile := range dataPack.Data {
		for i := range dataFile.Profiles {
			prof := &dataFile.Profiles[i]
			tailValues := prof.Data

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

// ---------- Glue handler ----------

func (d *HandlerDeps) handleGlue(ctx context.Context, t *asynq.Task) error {
	var payload GluePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal glue payload: %w", err)
	}

	log := d.Log.WithFields(logrus.Fields{
		"task_id":       t.ResultWriter().TaskID(),
		"prep_id":       payload.PrepID,
		"experiment_id": payload.ExperimentID,
	})

	tempDir, err := os.MkdirTemp("", "glue-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Download prepared zip from MinIO
	localZipPath := filepath.Join(tempDir, "prepared.zip")
	if err := d.Minio.DownloadFile(ctx, payload.PathToData, localZipPath); err != nil {
		return d.setGlueFailed(ctx, payload.PrepID, fmt.Sprintf("download prepared zip: %s", err.Error()))
	}

	// 2. Parse the zip into LicelPack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		return d.setGlueFailed(ctx, payload.PrepID, fmt.Sprintf("parse prepared zip: %s", err.Error()))
	}

	// 3. For each wavelength, perform glue
	for _, wvl := range payload.Wavelengths {
		if err := dataPack.Glue(wvl, payload.H1, payload.H2, payload.Polarization); err != nil {
			return d.setGlueFailed(ctx, payload.PrepID, fmt.Sprintf("glue failed for wavelength %.0f: %s", wvl, err.Error()))
		}
	}

	// 4. Save updated pack back to zip
	outputZipPath := filepath.Join(tempDir, "glued.zip")
	if err := dataPack.SaveToZip(outputZipPath); err != nil {
		return d.setGlueFailed(ctx, payload.PrepID, fmt.Sprintf("save glued zip: %s", err.Error()))
	}

	// 5. Upload glued zip to MinIO (overwrite)
	if err := d.Minio.UploadFile(ctx, payload.PathToData, outputZipPath, "application/zip"); err != nil {
		return d.setGlueFailed(ctx, payload.PrepID, fmt.Sprintf("upload glued zip: %s", err.Error()))
	}

	// 6. Update status to DoneStageTwo
	if err := d.PrepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:     payload.PrepID,
		Status: entity.PrepStatusDoneStageTwo,
	}); err != nil {
		return d.setGlueFailed(ctx, payload.PrepID, fmt.Sprintf("update status: %s", err.Error()))
	}

	log.WithFields(logrus.Fields{
		"wavelengths": payload.Wavelengths,
	}).Info("glue completed successfully")

	return nil
}

func (d *HandlerDeps) setGlueFailed(ctx context.Context, prepID uint, errMsg string) error {
	log := d.Log.WithField("prep_id", prepID).WithError(fmt.Errorf("%s", errMsg))
	log.Error("glue task failed")
	if err := d.PrepRepo.Update(ctx, &entity.PreparedExperiment{
		ID:       prepID,
		Status:   entity.PrepStatusFailed,
		ErrorMsg: errMsg,
	}); err != nil {
		log.WithError(err).Error("failed to set prepared experiment status to failed")
	}
	return fmt.Errorf("glue failed for prep %d: %s", prepID, errMsg)
}

// ---------- Visualize handler ----------

// namedProfile holds a profile together with its source file metadata.
type namedProfile struct {
	LicelProfile licelformat.LicelProfile
	StartTime    float64
	Filename     string
}

func (d *HandlerDeps) handleVisualize(ctx context.Context, t *asynq.Task) error {
	var payload VisualizePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal visualize payload: %w", err)
	}

	log := d.Log.WithFields(logrus.Fields{
		"task_id": t.ResultWriter().TaskID(),
		"prep_id": payload.PrepID,
	})
	taskID := t.ResultWriter().TaskID()

	// Write initial processing status
	_ = d.TaskStore.Set(ctx, taskID, &TaskResult{
		Status:    "processing",
		UpdatedAt: time.Now().Unix(),
	})

	outputType := payload.OutputType
	if outputType == "" {
		outputType = defaultOutputType
	}
	formula := payload.Formula
	if formula == "" {
		formula = "raw"
	}
	if formula != "raw" && formula != "rangecorr" && formula != "lograngecorr" {
		errMsg := fmt.Sprintf("unknown formula: %s (valid: raw, rangecorr, lograngecorr)", formula)
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: errMsg, UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("%s", errMsg)
	}

	// 1. Find PreparedExperiment to get ExperimentID
	prep, err := d.PrepRepo.FindByID(ctx, payload.PrepID)
	if err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("prepared experiment not found: %s", err.Error()), UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("prepared experiment not found: %w", err)
	}
	if prep.Status != entity.PrepStatusDone &&
		prep.Status != entity.PrepStatusDoneStageOne &&
		prep.Status != entity.PrepStatusDoneStageTwo {
		errMsg := fmt.Sprintf("prepared experiment %d is not ready (status: %s)", prep.ID, prep.Status)
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: errMsg, UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("%s", errMsg)
	}

	experimentID := prep.ExperimentID

	// 2. If not forced regenerate, try to find cached chart in DB
	if !payload.Regenerate {
		cached, err := d.ChartRepo.FindByParams(ctx, experimentID, payload.VizType, formula, payload.Wavelen, payload.Polarization, payload.IsPhoton, payload.Glued)
		if err != nil {
			log.WithError(err).Warn("failed to lookup cached chart, will regenerate")
		}
		if cached != nil {
			url, err := d.Minio.PresignedGetObject(ctx, cached.PathToObject, presignExpiry)
			if err != nil {
				_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("presigned url: %s", err.Error()), UpdatedAt: time.Now().Unix()})
				return fmt.Errorf("presigned url: %w", err)
			}
			_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "done", URL: url, UpdatedAt: time.Now().Unix()})
			log.WithField("path", cached.PathToObject).Info("returning cached chart")
			return nil
		}
	}

	// 3. Download prepared zip from Minio
	tempDir, err := os.MkdirTemp("", "visualize-*")
	if err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("create temp dir: %s", err.Error()), UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	localZipPath := filepath.Join(tempDir, "prepared.zip")
	if err := d.Minio.DownloadFile(ctx, prep.PathToData, localZipPath); err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("download prepared data: %s", err.Error()), UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("download prepared data: %w", err)
	}

	// 4. Parse the zip into LicelPack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("parse prepared zip: %s", err.Error()), UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("parse prepared zip: %w", err)
	}

	// 5. Extract matching profiles with time metadata
	profiles := extractProfiles(dataPack, payload.IsPhoton != 0, payload.Wavelen, payload.Polarization, payload.Glued != 0)
	if len(profiles) == 0 {
		errMsg := fmt.Sprintf("no profiles found for wavelen=%.0f isPhoton=%d polarization=%s", payload.Wavelen, payload.IsPhoton, payload.Polarization)
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: errMsg, UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("%s", errMsg)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].StartTime < profiles[j].StartTime
	})

	// 6. Generate visualization
	var result *visualize.Result
	switch payload.VizType {
	case "image":
		result, err = genHeatmap(profiles, outputType, formula)
	case "profile":
		result, err = genProfile(profiles, outputType, formula)
	default:
		errMsg := fmt.Sprintf("unknown visualization type: %s", payload.VizType)
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: errMsg, UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("%s", errMsg)
	}
	if err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: err.Error(), UpdatedAt: time.Now().Unix()})
		return err
	}

	// 7. Upload to Minio and save record to DB
	objectPath := fmt.Sprintf("experiments/%d/images/%s-%.0f-%d-%s-%d-%s.%s",
		experimentID, payload.VizType, payload.Wavelen, payload.Glued, payload.Polarization, payload.IsPhoton, formula, outputType)

	if err := d.Minio.UploadBytes(ctx, objectPath, result.Body, result.ContentType); err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("upload chart to minio: %s", err.Error()), UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("upload chart to minio: %w", err)
	}

	chart := &entity.ExperimentChart{
		ExperimentID: experimentID,
		ChartType:    payload.VizType,
		Formula:      formula,
		Wavelen:      payload.Wavelen,
		Polarization: payload.Polarization,
		IsPhoton:     payload.IsPhoton,
		Glued:        payload.Glued,
		PathToObject: objectPath,
	}
	if err := d.ChartRepo.Create(ctx, chart); err != nil {
		log.WithError(err).Warn("failed to save experiment chart record (chart is still uploaded)")
	}

	url, err := d.Minio.PresignedGetObject(ctx, objectPath, presignExpiry)
	if err != nil {
		_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "failed", Error: fmt.Sprintf("presigned url: %s", err.Error()), UpdatedAt: time.Now().Unix()})
		return fmt.Errorf("presigned url: %w", err)
	}

	_ = d.TaskStore.Set(ctx, taskID, &TaskResult{Status: "done", URL: url, UpdatedAt: time.Now().Unix()})
	log.Info("visualization completed successfully")
	return nil
}

// extractProfiles walks all files in the pack and collects matching profiles.
func extractProfiles(dataPack *licelformat.LicelPack, isPhoton bool, wavelen float64, polarization string, glued bool) []namedProfile {
	var result []namedProfile
	for fname, licf := range dataPack.Data {
		if glued {
			for _, prof := range licf.Profiles {
				if prof.IsGlued() && prof.Wavelength == wavelen && prof.Polarization == polarization {
					result = append(result, namedProfile{
						LicelProfile: prof,
						StartTime:    float64(licf.MeasurementStartTime.Unix()),
						Filename:     fname,
					})
				}
			}
		} else {
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
	}
	return result
}

func genHeatmap(profiles []namedProfile, outputType string, formula string) (*visualize.Result, error) {
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

	binWidth := profiles[0].LicelProfile.BinWidth
	if binWidth <= 0 {
		binWidth = 7.5
	}

	nTime := len(profiles)
	timeLabels := make([]string, nTime)
	for i, p := range profiles {
		timeLabels[i] = visualize.FormatTimeHHMM(int64(p.StartTime))
	}

	distanceLabels := make([]string, maxBins)
	for i := 0; i < maxBins; i++ {
		distanceLabels[i] = fmt.Sprintf("%.0f", float64(i)*binWidth)
	}

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
		return visualize.HeatmapToPlotly(timeLabels, distanceLabels, zData, titleSuffix)
	case "png":
		return visualize.HeatmapToPNG(timeLabels, distanceLabels, zData, titleSuffix)
	default:
		return visualize.HeatmapToSVG(timeLabels, distanceLabels, zData, titleSuffix)
	}
}

func genProfile(profiles []namedProfile, outputType string, formula string) (*visualize.Result, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("no profiles to average")
	}

	binWidth := profiles[0].LicelProfile.BinWidth
	if binWidth <= 0 {
		binWidth = 7.5
	}

	maxLen := 0
	for _, p := range profiles {
		if len(p.LicelProfile.Data) > maxLen {
			maxLen = len(p.LicelProfile.Data)
		}
	}

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

	distance := make([]float64, maxLen)
	for i := 0; i < maxLen; i++ {
		distance[i] = float64(i) * binWidth
	}

	switch outputType {
	case "json":
		return visualize.ProfileToPlotly(distance, avgData, titleSuffix)
	case "png":
		return visualize.ProfileToPNG(distance, avgData, titleSuffix)
	default:
		return visualize.ProfileToSVG(distance, avgData, titleSuffix)
	}
}
