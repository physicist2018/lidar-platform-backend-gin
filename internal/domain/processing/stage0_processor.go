package processing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/physicist2018/lidar-platform-go/internal/domain/entity"
	"github.com/physicist2018/lidar-platform-go/internal/domain/repository"
)

// Stage0Processor implements the stage0 algorithm: background subtraction and analog/digital gluing.
type Stage0Processor struct {
	lidarPackRepo    repository.LidarPackRepository
	processedSigRepo repository.ProcessedSignalRepository
	expRepo          repository.ExperimentRepository
	log              *logrus.Logger
}

var _ Processor = (*Stage0Processor)(nil)

func NewStage0Processor(
	lidarPackRepo repository.LidarPackRepository,
	processedSigRepo repository.ProcessedSignalRepository,
	expRepo repository.ExperimentRepository,
	log *logrus.Logger,
) *Stage0Processor {
	return &Stage0Processor{
		lidarPackRepo:    lidarPackRepo,
		processedSigRepo: processedSigRepo,
		expRepo:          expRepo,
		log:              log,
	}
}

func (p *Stage0Processor) Name() string {
	return "stage0"
}

func (p *Stage0Processor) Execute(ctx context.Context, run *entity.ProcessingRun) error {
	log := p.log.WithFields(logrus.Fields{
		"algorithm":     run.Algorithm,
		"processing_id": run.ID,
		"experiment_id": run.ExperimentID,
	})

	log.Info("starting stage0 processing")

	// 1. Parse parameters
	var params entity.Stage0Params
	if err := json.Unmarshal(run.Params, &params); err != nil {
		return fmt.Errorf("parse stage0 params: %w", err)
	}

	// 2. Load original profiles from the experiment's main data pack
	profiles, err := p.lidarPackRepo.GetProfilesByExperimentID(ctx, run.ExperimentID)
	if err != nil {
		return fmt.Errorf("load profiles: %w", err)
	}
	log.WithField("profile_count", len(profiles)).Info("loaded original profiles")

	// 3. Background subtraction
	processed, err := p.subtractBackground(ctx, run.ExperimentID, profiles, &params.Background)
	if err != nil {
		return fmt.Errorf("background subtraction: %w", err)
	}
	log.Info("background subtraction completed")

	// 4. Glue analog/digital channels
	if len(params.Glue) > 0 {
		processed, err = p.glueChannels(processed, params.Glue)
		if err != nil {
			return fmt.Errorf("glue channels: %w", err)
		}
		log.Info("channel gluing completed")
	}

	// 5. Save processed signals to DB
	signals := make([]entity.ProcessedSignal, len(processed))
	for i, prof := range processed {
		signals[i] = entity.ProcessedSignal{
			ProcessingRunID:   run.ID,
			OriginalProfileID: prof.ID,
			Wavelength:        prof.Wavelength,
			Polarization:      prof.Polarization,
			IsPhoton:          prof.IsPhoton,
			Signal:            prof.Signal,
		}
	}

	if err := p.processedSigRepo.BatchCreate(ctx, signals); err != nil {
		return fmt.Errorf("save processed signals: %w", err)
	}
	log.WithField("saved_count", len(signals)).Info("processed signals saved")

	return nil
}

// subtractBackground applies background subtraction based on the configured method.
func (p *Stage0Processor) subtractBackground(
	ctx context.Context,
	experimentID uint,
	profiles []entity.LidarProfile,
	bgr *entity.BackgroundParams,
) ([]entity.LidarProfile, error) {
	if bgr == nil {
		return profiles, nil
	}

	result := make([]entity.LidarProfile, len(profiles))
	copy(result, profiles)

	switch bgr.Type {
	case "avgtail", "medtail":
		tailFn := avg
		if bgr.Type == "medtail" {
			tailFn = median
		}
		for i := range result {
			prof := &result[i]
			tailValues := prof.Signal
			if prof.BinWidth > 0 && bgr.BgrFrom > 0 {
				startIdx := int(math.Ceil(bgr.BgrFrom / prof.BinWidth))
				if startIdx < 0 {
					startIdx = 0
				}
				if startIdx < len(prof.Signal) {
					tailValues = prof.Signal[startIdx:]
				}
			}
			bgValue := tailFn(tailValues)
			newSig := make([]float64, len(prof.Signal))
			for j, v := range prof.Signal {
				newSig[j] = v - bgValue
			}
			prof.Signal = newSig
		}

	case "file":
		// Load experiment to get BgrFileID
		exp, err := p.expRepo.FindByID(ctx, experimentID)
		if err != nil {
			return nil, fmt.Errorf("load experiment: %w", err)
		}
		if exp.BgrFileID == nil {
			return nil, fmt.Errorf("experiment %d has no BGR file assigned", experimentID)
		}

		// Load BGR profiles from the file
		bgrProfiles, err := p.lidarPackRepo.GetProfilesByFileID(ctx, *exp.BgrFileID)
		if err != nil {
			return nil, fmt.Errorf("load bgr profiles: %w", err)
		}

		// Build lookup map: (wavelength, polarization, isPhoton) → BGR profile
		bgrMap := make(map[string]*entity.LidarProfile)
		for i := range bgrProfiles {
			key := fmt.Sprintf("%.1f|%s|%v", bgrProfiles[i].Wavelength, bgrProfiles[i].Polarization, bgrProfiles[i].IsPhoton)
			bgrMap[key] = &bgrProfiles[i]
		}

		for i := range result {
			prof := &result[i]
			key := fmt.Sprintf("%.1f|%s|%v", prof.Wavelength, prof.Polarization, prof.IsPhoton)
			bgrProf, ok := bgrMap[key]
			if !ok {
				p.log.WithFields(logrus.Fields{
					"wavelength":   prof.Wavelength,
					"polarization": prof.Polarization,
					"is_photon":    prof.IsPhoton,
				}).Warn("no matching BGR profile found, skipping subtraction")
				continue
			}

			newSig := make([]float64, len(prof.Signal))
			minLen := len(prof.Signal)
			if len(bgrProf.Signal) < minLen {
				minLen = len(bgrProf.Signal)
			}
			for j := 0; j < minLen; j++ {
				newSig[j] = prof.Signal[j] - bgrProf.Signal[j]
			}
			// Copy remaining samples beyond BGR length
			for j := minLen; j < len(prof.Signal); j++ {
				newSig[j] = prof.Signal[j]
			}
			prof.Signal = newSig
		}

	default:
		return nil, fmt.Errorf("unknown background type: %s", bgr.Type)
	}

	return result, nil
}

// glueChannels performs analog/digital channel gluing for specified parameters.
// Returns new profiles — modified originals plus added glued profiles.
func (p *Stage0Processor) glueChannels(
	profiles []entity.LidarProfile,
	glueParams []entity.GlueParam,
) ([]entity.LidarProfile, error) {
	// Build a map of profiles by (wavelength, polarization, isPhoton)
	profileMap := make(map[string]*entity.LidarProfile)
	for i := range profiles {
		key := channelKey(profiles[i].Wavelength, profiles[i].Polarization, profiles[i].IsPhoton)
		profileMap[key] = &profiles[i]
	}

	for _, gp := range glueParams {
		analogKey := channelKey(gp.Wavelength, gp.Polarization, false)
		digitalKey := channelKey(gp.Wavelength, gp.Polarization, true)

		analogProf, okAnalog := profileMap[analogKey]
		digitalProf, okDigital := profileMap[digitalKey]

		if !okAnalog || !okDigital {
			// Try with empty polarization
			analogKey2 := channelKey(gp.Wavelength, "", false)
			digitalKey2 := channelKey(gp.Wavelength, "", true)

			if !okAnalog {
				analogProf, okAnalog = profileMap[analogKey2]
			}
			if !okDigital {
				digitalProf, okDigital = profileMap[digitalKey2]
			}

			if !okAnalog || !okDigital {
				p.log.WithFields(logrus.Fields{
					"wavelength":   gp.Wavelength,
					"polarization": gp.Polarization,
				}).Warn("cannot glue — missing analog or digital channel")
				continue
			}
		}

		// Calculate overlap indices from altitude range [r0, r1]
		binWidth := analogProf.BinWidth
		if binWidth <= 0 {
			binWidth = digitalProf.BinWidth
		}
		if binWidth <= 0 {
			continue
		}

		r0Idx := int(math.Ceil(gp.R0 / binWidth))
		r1Idx := int(math.Ceil(gp.R1 / binWidth))

		// Clamp to profile bounds
		maxLen := len(analogProf.Signal)
		if len(digitalProf.Signal) < maxLen {
			maxLen = len(digitalProf.Signal)
		}
		if r0Idx >= maxLen {
			r0Idx = maxLen - 1
		}
		if r1Idx > maxLen {
			r1Idx = maxLen
		}
		if r0Idx >= r1Idx {
			continue
		}

		// Compute scaling factor: k = mean(analog[r0:r1]) / mean(digital[r0:r1])
		meanAnalog := mean(analogProf.Signal[r0Idx:r1Idx])
		meanDigital := mean(digitalProf.Signal[r0Idx:r1Idx])

		if meanDigital == 0 {
			p.log.Warn("digital mean is zero, skipping glue")
			continue
		}

		k := meanAnalog / meanDigital

		// Build the glued signal
		gluedLen := maxLen
		gluedSig := make([]float64, gluedLen)

		switch gp.ScaleTo {
		case "analog":
			// analog[0:r0] + digital_scaled[r0:]
			// The analog channel retains its original low-altitude data.
			// The digital channel is scaled UP to match analog level.
			for j := 0; j < r0Idx && j < len(analogProf.Signal); j++ {
				gluedSig[j] = analogProf.Signal[j]
			}
			for j := r0Idx; j < gluedLen && j < len(digitalProf.Signal); j++ {
				gluedSig[j] = digitalProf.Signal[j] * k
			}
			// Replace digital profile signal with the glued composite
			digitalProf.Signal = gluedSig

		case "digital":
			// analog_scaled[0:r0] + digital[r0:]
			// The analog channel is scaled DOWN to match digital level.
			// The digital channel retains its original high-altitude data.
			for j := 0; j < r0Idx && j < len(analogProf.Signal); j++ {
				gluedSig[j] = analogProf.Signal[j] / k
			}
			for j := r0Idx; j < gluedLen && j < len(digitalProf.Signal); j++ {
				gluedSig[j] = digitalProf.Signal[j]
			}
			// Replace analog profile signal with the glued composite
			analogProf.Signal = gluedSig
		}
	}

	// Return all profiles (modified ones already updated in-place in the slice)
	return profiles, nil
}

// channelKey creates a lookup key for a profile.
func channelKey(wavelength float64, polarization string, isPhoton bool) string {
	return fmt.Sprintf("%.1f|%s|%v", wavelength, polarization, isPhoton)
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

func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}
