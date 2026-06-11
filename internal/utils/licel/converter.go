package licel

import (
	"github.com/physicist2018/licelfile/v2/licelformat"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
)

// FromLicelPack converts a licelformat.LicelPack (from the external library)
// into the internal domain entity.LidarPack with full hierarchy.
// pack.Data keys are used as filenames.
func FromLicelPack(experimentID uint, pack *licelformat.LicelPack) *entity.LidarPack {
	lp := &entity.LidarPack{
		ExperimentID: experimentID,
		Files:        make([]entity.LidarFile, 0, len(pack.Data)),
	}

	if !pack.StartTime.IsZero() {
		st := pack.StartTime
		lp.StartTime = &st
	}
	if !pack.StopTime.IsZero() {
		st := pack.StopTime
		lp.StopTime = &st
	}

	for fname, lf := range pack.Data {
		displayName := fname

		lidarFile := entity.LidarFile{
			Filename:     displayName,
			Site:         lf.MeasurementSite,
			Altitude:     lf.AltitudeAboveSeaLevel,
			Longitude:    lf.Longitude,
			Latitude:     lf.Latitude,
			Zenith:       lf.Zenith,
			Laser1NShots: lf.Laser1NShots,
			Laser1Freq:   lf.Laser1Freq,
			Laser2NShots: lf.Laser2NShots,
			Laser2Freq:   lf.Laser2Freq,
			Laser3NShots: lf.Laser3NShots,
			Laser3Freq:   lf.Laser3Freq,
			NDatasets:    lf.NDatasets,
			Profiles:     make([]entity.LidarProfile, 0, len(lf.Profiles)),
		}

		if !lf.MeasurementStartTime.IsZero() {
			t := lf.MeasurementStartTime
			lidarFile.StartTime = &t
		}
		if !lf.MeasurementStopTime.IsZero() {
			t := lf.MeasurementStopTime
			lidarFile.StopTime = &t
		}

		for _, prof := range lf.Profiles {
			reserved := make([]int, len(prof.Reserved))
			for i, v := range prof.Reserved {
				reserved[i] = v
			}

			signal := make([]float64, len(prof.Data))
			copy(signal, prof.Data)

			lidarProfile := entity.LidarProfile{
				Active:       prof.Active,
				IsPhoton:     prof.Photon,
				LaserType:    prof.LaserType,
				NDataPoints:  prof.NDataPoints,
				Reserved:     reserved,
				HighVoltage:  prof.HighVoltage,
				BinWidth:     prof.BinWidth,
				Wavelength:   prof.Wavelength,
				Polarization: prof.Polarization,
				BinShift:     prof.BinShift,
				DecBinShift:  prof.DecBinShift,
				AdcBits:      prof.AdcBits,
				NShots:       prof.NShots,
				DiscrLevel:   prof.DiscrLevel,
				DeviceID:     prof.DeviceID,
				NCrate:       prof.NCrate,
				Signal:       signal,
			}

			lidarFile.Profiles = append(lidarFile.Profiles, lidarProfile)
		}

		lp.Files = append(lp.Files, lidarFile)
	}

	return lp
}
