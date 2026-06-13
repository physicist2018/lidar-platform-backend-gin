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

	// 4. Crop profiles (truncate data above crop_from)
	processed = p.cropProfiles(processed, params.Crop.CropFrom)
	log.Info("cropping completed")

	// 5. Glue analog/digital channels — creates new profiles with DeviceID="BG"
	if len(params.Glue) > 0 {
		processed, err = p.glueChannels(processed, params.Glue)
		if err != nil {
			return fmt.Errorf("glue channels: %w", err)
		}
		log.Info("channel gluing completed")
	}

	// 6. Save processed signals to DB
	signals := make([]entity.ProcessedSignal, len(processed))
	for i, prof := range processed {
		signals[i] = entity.ProcessedSignal{
			ProcessingRunID:   run.ID,
			OriginalProfileID: prof.ID,
			FileID:            prof.FileID,
			Wavelength:        prof.Wavelength,
			Polarization:      prof.Polarization,
			IsPhoton:          prof.IsPhoton,
			DeviceID:          prof.DeviceID,
			BinWidth:          prof.BinWidth,
			NDataPoints:       prof.NDataPoints,
			Signal:            prof.Signal,
		}
	}

	if err := p.processedSigRepo.BatchCreate(ctx, signals); err != nil {
		return fmt.Errorf("save processed signals: %w", err)
	}
	log.WithField("saved_count", len(signals)).Info("processed signals saved")

	return nil
}

// cropProfiles truncates all profiles so that data above cropFrom is removed.
// If cropFrom <= 0, profiles are returned unchanged.
func (p *Stage0Processor) cropProfiles(profiles []entity.LidarProfile, cropFrom float64) []entity.LidarProfile {
	if cropFrom <= 0 {
		return profiles
	}

	result := make([]entity.LidarProfile, len(profiles))
	copy(result, profiles)

	for i := range result {
		prof := &result[i]
		if prof.BinWidth <= 0 {
			continue
		}
		maxIdx := int(math.Ceil(cropFrom / prof.BinWidth))
		if maxIdx <= 0 || maxIdx >= len(prof.Signal) {
			continue
		}
		prof.Signal = prof.Signal[:maxIdx]
		prof.NDataPoints = len(prof.Signal)
	}
	return result
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
// For each file in the experiment, it finds pairs of profiles with matching
// (wavelength, polarization), where one has DeviceID="BT" (analog) and the other
// has DeviceID="BC" (digital). These are glued together into a new profile
// with DeviceID="BG".
// Returns profiles + newly created glued profiles.
// Original profiles are NOT modified.
func (p *Stage0Processor) glueChannels(
	profiles []entity.LidarProfile,
	glueParams []entity.GlueParam,
) ([]entity.LidarProfile, error) {
	// Group profiles by FileID
	type fileGroup struct {
		profiles []entity.LidarProfile
		indices  []int // indices into the original profiles slice
	}

	fileMap := make(map[uint]*fileGroup)
	for i := range profiles {
		prof := &profiles[i]
		fg, ok := fileMap[prof.FileID]
		if !ok {
			fg = &fileGroup{}
			fileMap[prof.FileID] = fg
		}
		fg.profiles = append(fg.profiles, *prof)
		fg.indices = append(fg.indices, i)
	}

	// deviceKey creates a lookup key: "wavelength|polarization|deviceID"
	deviceKey := func(wavelength float64, polarization string, deviceID string) string {
		return fmt.Sprintf("%.1f|%s|%s", wavelength, polarization, deviceID)
	}

	var newProfiles []entity.LidarProfile

	for _, fg := range fileMap {
		// Build lookup: deviceKey → index within this file group
		lookup := make(map[string]int) // key → index in fg.profiles
		for i := range fg.profiles {
			key := deviceKey(fg.profiles[i].Wavelength, fg.profiles[i].Polarization, fg.profiles[i].DeviceID)
			lookup[key] = i
		}

		for _, gp := range glueParams {
			btKey := deviceKey(gp.Wavelength, gp.Polarization, "BT")
			bcKey := deviceKey(gp.Wavelength, gp.Polarization, "BC")

			btIdx, okBT := lookup[btKey]
			bcIdx, okBC := lookup[bcKey]

			if !okBT || !okBC {
				// Try with empty polarization
				if !okBT {
					btKey2 := deviceKey(gp.Wavelength, "", "BT")
					btIdx, okBT = lookup[btKey2]
				}
				if !okBC {
					bcKey2 := deviceKey(gp.Wavelength, "", "BC")
					bcIdx, okBC = lookup[bcKey2]
				}

				if !okBT || !okBC {
					p.log.WithFields(logrus.Fields{
						"wavelength":   gp.Wavelength,
						"polarization": gp.Polarization,
					}).Warn("cannot glue — missing BT (analog) or BC (digital) profile in file")
					continue
				}
			}

			btProf := &fg.profiles[btIdx]
			bcProf := &fg.profiles[bcIdx]

			// Calculate overlap indices from altitude range [r0, r1]
			binWidth := btProf.BinWidth
			if binWidth <= 0 {
				binWidth = bcProf.BinWidth
			}
			if binWidth <= 0 {
				continue
			}

			r0Idx := int(math.Ceil(gp.R0 / binWidth))
			r1Idx := int(math.Ceil(gp.R1 / binWidth))

			// Clamp to profile bounds
			maxLen := len(btProf.Signal)
			if len(bcProf.Signal) < maxLen {
				maxLen = len(bcProf.Signal)
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

			// Compute scaling factor: k = mean(BT[r0:r1]) / mean(BC[r0:r1])
			meanBT := mean(btProf.Signal[r0Idx:r1Idx])
			meanBC := mean(bcProf.Signal[r0Idx:r1Idx])

			if meanBC == 0 {
				p.log.Warn("BC (digital) mean is zero, skipping glue")
				continue
			}

			k := meanBT / meanBC

			// Build the glued signal
			gluedSig := make([]float64, maxLen)

			// Determine which profile to copy metadata from
			// "analog" → BT; "digital" → BC
			var template *entity.LidarProfile
			if gp.ScaleTo == "analog" {
				template = btProf
			} else {
				template = bcProf
			}

			switch gp.ScaleTo {
			case "analog":
				// BT[0:r0] + BC_scaled[r0:]
				for j := 0; j < r0Idx && j < len(btProf.Signal); j++ {
					gluedSig[j] = btProf.Signal[j]
				}
				for j := r0Idx; j < maxLen && j < len(bcProf.Signal); j++ {
					gluedSig[j] = bcProf.Signal[j] * k
				}

			case "digital":
				// BT_scaled[0:r0] + BC[r0:]
				for j := 0; j < r0Idx && j < len(btProf.Signal); j++ {
					gluedSig[j] = btProf.Signal[j] / k
				}
				for j := r0Idx; j < maxLen && j < len(bcProf.Signal); j++ {
					gluedSig[j] = bcProf.Signal[j]
				}
			}

			// Create a new profile with DeviceID="BG"
			gluedProfile := entity.LidarProfile{
				ID:           0, // will be ignored on save (maps to processed_signals.original_profile_id)
				FileID:       template.FileID,
				Active:       template.Active,
				IsPhoton:     template.IsPhoton,
				LaserType:    template.LaserType,
				NDataPoints:  template.NDataPoints,
				Reserved:     template.Reserved,
				HighVoltage:  template.HighVoltage,
				BinWidth:     template.BinWidth,
				Wavelength:   template.Wavelength,
				Polarization: template.Polarization,
				BinShift:     template.BinShift,
				DecBinShift:  template.DecBinShift,
				AdcBits:      template.AdcBits,
				NShots:       template.NShots,
				DiscrLevel:   template.DiscrLevel,
				DeviceID:     "BG",
				NCrate:       template.NCrate,
				Signal:       gluedSig,
			}
			newProfiles = append(newProfiles, gluedProfile)

			p.log.WithFields(logrus.Fields{
				"file_id":      fg.profiles[0].FileID,
				"wavelength":   gp.Wavelength,
				"polarization": gp.Polarization,
				"scale_to":     gp.ScaleTo,
				"k":            k,
				"len":          maxLen,
			}).Info("glued profile created")
		}
	}

	return append(profiles, newProfiles...), nil
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
