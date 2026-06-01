package implementation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"

	"github.com/kshmirko/lidar-platform-go/internal/domain/entity"
	"github.com/kshmirko/lidar-platform-go/internal/domain/repository"
	"github.com/kshmirko/lidar-platform-go/internal/domain/usecase"
	"github.com/kshmirko/lidar-platform-go/internal/infrastructure/storage"
)

const (
	defaultOutputType = "svg"
)

type visualizePreparedExperimentUseCaseImpl struct {
	prepRepo repository.PreparedExperimentRepository
	minio    *storage.MinioClient
	log      *logrus.Logger
}

var _ usecase.VisualizePreparedExperimentUseCase = (*visualizePreparedExperimentUseCaseImpl)(nil)

func NewVisualizePreparedExperimentUseCaseImpl(
	prepRepo repository.PreparedExperimentRepository,
	minio *storage.MinioClient,
	log *logrus.Logger,
) *visualizePreparedExperimentUseCaseImpl {
	return &visualizePreparedExperimentUseCaseImpl{
		prepRepo: prepRepo,
		minio:    minio,
		log:      log,
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
	isPhoton bool,
	polarization string,
	vizType string,
	outputType string,
	formula string,
) (*usecase.VisualizeResult, error) {
	if outputType == "" {
		outputType = defaultOutputType
	}
	if formula == "" {
		formula = "raw"
	}
	if formula != "raw" && formula != "rangecorr" && formula != "lograngecorr" {
		return nil, fmt.Errorf("unknown formula: %s (valid: raw, rangecorr, lograngecorr)", formula)
	}

	// 1. Find PreparedExperiment
	prep, err := u.prepRepo.FindByID(ctx, prepID)
	if err != nil {
		return nil, fmt.Errorf("prepared experiment not found: %w", err)
	}
	if prep.Status != entity.PrepStatusDone {
		return nil, fmt.Errorf("prepared experiment %d is not ready (status: %s)", prep.ID, prep.Status)
	}

	// 2. Download prepared zip from Minio
	tempDir, err := os.MkdirTemp("", "visualize-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	localZipPath := filepath.Join(tempDir, "prepared.zip")
	if err := u.minio.DownloadFile(ctx, prep.PathToData, localZipPath); err != nil {
		return nil, fmt.Errorf("download prepared data: %w", err)
	}

	// 3. Parse the zip into LicelPack
	dataPack, err := licelformat.NewLicelPackFromZip(localZipPath)
	if err != nil {
		return nil, fmt.Errorf("parse prepared zip: %w", err)
	}

	// 4. Extract matching profiles with time metadata
	profiles := u.extractProfiles(dataPack, isPhoton, wavelen, polarization)
	if len(profiles) == 0 {
		return nil, fmt.Errorf(
			"no profiles found for wavelen=%.0f photon=%v polarization=%s",
			wavelen, isPhoton, polarization,
		)
	}

	// Sort by start time
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].StartTime < profiles[j].StartTime
	})

	// 5. Generate visualization
	switch vizType {
	case "image":
		return u.genHeatmap(profiles, outputType, formula)
	case "profile":
		return u.genProfile(profiles, outputType, formula)
	default:
		return nil, fmt.Errorf("unknown visualization type: %s", vizType)
	}
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

// genHeatmap builds a heatmap: X = time, Y = distance, Z = signal intensity.
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
		timeLabels[i] = formatTimeHHMM(int64(p.StartTime))
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
		applyFormula(row, formula, binWidth)
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
		return u.heatmapToPlotly(timeLabels, distanceLabels, zData, titleSuffix)
	case "png":
		return u.heatmapToPNG(timeLabels, distanceLabels, zData, titleSuffix)
	default:
		return u.heatmapToSVG(timeLabels, distanceLabels, zData, titleSuffix)
	}
}

// genProfile builds an averaged XY profile.
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
	applyFormula(avgData, formula, binWidth)

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
		return u.profileToPlotly(distance, avgData, titleSuffix)
	case "png":
		return u.profileToPNG(distance, avgData, titleSuffix)
	default:
		return u.profileToSVG(distance, avgData, titleSuffix)
	}
}

// ========================= SVG generation =========================

// formatTimeHHMM formats a Unix timestamp as HH:MM in local time.
func formatTimeHHMM(unix int64) string {
	t := time.Unix(unix, 0)
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

func (u *visualizePreparedExperimentUseCaseImpl) heatmapToSVG(
	timeLabels []string,
	distanceLabels []string,
	zData [][]float64,
	titleSuffix string,
) (*usecase.VisualizeResult, error) {
	width, height := 900, 650
	marginLeft, marginRight, marginTop, marginBottom := 70.0, 55.0, 40.0, 80.0
	plotW := float64(width) - marginLeft - marginRight
	plotH := float64(height) - marginTop - marginBottom

	nTime := len(zData)          // columns → X = time
	nDist := len(distanceLabels) // rows → Y = distance
	if nTime == 0 || nDist == 0 {
		return nil, fmt.Errorf("empty data for heatmap")
	}

	// Transpose: zT[dist][time]
	zT := make([][]float64, nDist)
	for d := 0; d < nDist; d++ {
		row := make([]float64, nTime)
		for t := 0; t < nTime; t++ {
			if d < len(zData[t]) {
				row[t] = zData[t][d]
			}
		}
		zT[d] = row
	}

	cellW := plotW / float64(nTime)
	cellH := plotH / float64(nDist)

	// Collect values and compute 5th–95th percentile range for color scaling.
	// This clips outliers and improves contrast in the heatmap.
	allVals := make([]float64, 0, nTime*nDist)
	for _, row := range zT {
		allVals = append(allVals, row...)
	}
	sort.Float64s(allVals)
	zMin := percentile(allVals, 0.05)
	zMax := percentile(allVals, 0.95)
	if zMax == zMin {
		zMax = zMin + 1
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`, width, height))
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="white"/>`, width, height))

	// Title
	sb.WriteString(fmt.Sprintf(
		`<text x="%d" y="%d" text-anchor="middle" font-size="16" font-family="sans-serif">Lidar Heatmap%s</text>`,
		width/2, 25, titleSuffix,
	))

	// Draw cells → Y = distance (row d), flipped: d=0 at bottom, d=nDist-1 at top
	for d := 0; d < nDist; d++ {
		for t := 0; t < nTime; t++ {
			val := zT[d][t]
			r, g, bb := heatmapColor((val - zMin) / (zMax - zMin))
			x := marginLeft + float64(t)*cellW
			y := marginTop + float64(nDist-1-d)*cellH
			sb.WriteString(fmt.Sprintf(
				`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="rgb(%d,%d,%d)"/>`,
				x, y, cellW+1, cellH+1, r, g, bb,
			))
		}
	}

	// Grid lines (horizontal & vertical, dashed)
	numHGrid := 8
	for k := 0; k <= numHGrid; k++ {
		y := marginTop + float64(k)*plotH/float64(numHGrid)
		sb.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#ddd" stroke-width="0.5" stroke-dasharray="3,3"/>`,
			marginLeft, y, marginLeft+plotW, y,
		))
	}
	numVGrid := minInt(10, nTime-1)
	for k := 0; k <= numVGrid; k++ {
		x := marginLeft + float64(k)*plotW/float64(numVGrid)
		sb.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#ddd" stroke-width="0.5" stroke-dasharray="3,3"/>`,
			x, marginTop, x, marginTop+plotH,
		))
	}

	// Y-axis (distance) labels — flipped: k=0 → bottom, k=max → top
	numYLabels := 8
	for k := 0; k <= numYLabels; k++ {
		idx := int(float64(k) / float64(numYLabels) * float64(nDist-1))
		if idx >= nDist {
			idx = nDist - 1
		}
		y := marginTop + float64(nDist-1-idx)*cellH + cellH/2
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="10" font-family="sans-serif">%s m</text>`,
			marginLeft-5, y+4, distanceLabels[idx],
		))
	}

	// X-axis (time) labels
	numXLabels := minInt(10, nTime-1)
	for k := 0; k <= numXLabels; k++ {
		idx := int(float64(k) / float64(numXLabels) * float64(nTime-1))
		if idx >= nTime {
			idx = nTime - 1
		}
		x := marginLeft + float64(idx)*cellW + cellW/2
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="9" font-family="sans-serif">%s</text>`,
			x, marginTop+plotH+15, timeLabels[idx],
		))
	}

	// Axis lines
	sb.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="black" stroke-width="1"/>`,
		marginLeft, marginTop, marginLeft, marginTop+plotH,
	))
	sb.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="black" stroke-width="1"/>`,
		marginLeft, marginTop+plotH, marginLeft+plotW, marginTop+plotH,
	))

	// Axis titles
	sb.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-family="sans-serif" transform="rotate(-90,%.1f,%.1f)">Distance, m</text>`,
		15.0, marginTop+plotH/2, 15.0, marginTop+plotH/2,
	))
	sb.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-family="sans-serif">Time, HH:MM</text>`,
		marginLeft+plotW/2, marginTop+plotH+40,
	))

	// Color bar
	barX, barY, barW, barH := marginLeft+plotW+10, marginTop, 15.0, plotH
	for k := 0; k < 100; k++ {
		t := float64(k) / 100.0
		r, g, bb := heatmapColor(t)
		y := barY + barH - (t * barH)
		sb.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="rgb(%d,%d,%d)"/>`,
			barX, y, barW, barH/100+1, r, g, bb,
		))
	}
	sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="none" stroke="black"/>`, barX, barY, barW, barH))

	// Colorbar ticks (≥5 evenly spaced)
	numTicks := 5
	tickLen := 6.0
	for k := 0; k <= numTicks; k++ {
		frac := 1.0 - float64(k)/float64(numTicks) // 0=bottom, 1=top
		tickY := barY + frac*barH
		val := zMin + (1.0-frac)*(zMax-zMin)
		sb.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="black" stroke-width="1"/>`,
			barX-tickLen, tickY, barX, tickY,
		))
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="10" font-family="sans-serif">%.2e</text>`,
			barX-tickLen-3, tickY+4, val,
		))
	}

	sb.WriteString(`</svg>`)

	return &usecase.VisualizeResult{
		ContentType: "image/svg+xml",
		Body:        []byte(sb.String()),
	}, nil
}

// minInt returns the smaller of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// percentile returns the value at the given percentile (0..1) using linear interpolation.
// The input slice must be sorted in ascending order.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	k := p * float64(len(sorted)-1)
	f := math.Floor(k)
	c := math.Ceil(k)
	if f == c {
		return sorted[int(k)]
	}
	return sorted[int(f)]*(c-k) + sorted[int(c)]*(k-f)
}

// applyFormula transforms profile data in-place according to the formula:
// "raw" — P (no change), "rangecorr" — P × r², "lograngecorr" — log₁₀(P × r²).
func applyFormula(data []float64, formula string, binWidth float64) {
	switch formula {
	case "rangecorr":
		for i, v := range data {
			r := float64(i) * binWidth / 1000.0
			if v < 0 {
				v = 0
			}
			data[i] = v * r * r
		}
	case "lograngecorr":
		for i, v := range data {
			r := float64(i) * binWidth / 1000.0
			val := v * r * r
			if val > 0 {
				data[i] = math.Log10(val)
			}
		}
	}
}

// heatmapColor returns RGB for a value in [0, 1] using a blue-to-red ramp.
func heatmapColor(t float64) (r, g, b int) {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	// Blue (0,0,255) → Cyan (0,255,255) → Green (0,255,0) → Yellow (255,255,0) → Red (255,0,0)
	if t < 0.25 {
		s := t / 0.25
		return 0, int(255 * s), 255
	} else if t < 0.5 {
		s := (t - 0.25) / 0.25
		return 0, 255, int(255 * (1 - s))
	} else if t < 0.75 {
		s := (t - 0.5) / 0.25
		return int(255 * s), 255, 0
	} else {
		s := (t - 0.75) / 0.25
		return 255, int(255 * (1 - s)), 0
	}
}

func (u *visualizePreparedExperimentUseCaseImpl) profileToSVG(
	distance []float64,
	data []float64,
	titleSuffix string,
) (*usecase.VisualizeResult, error) {
	width, height := 800, 500
	marginLeft, marginRight, marginTop, marginBottom := 70.0, 30.0, 40.0, 60.0
	plotW := float64(width) - marginLeft - marginRight
	plotH := float64(height) - marginTop - marginBottom

	n := len(data)
	if n == 0 {
		return nil, fmt.Errorf("empty profile data")
	}

	xMin, xMax := distance[0], distance[n-1]
	yMin, yMax := data[0], data[0]
	for _, v := range data {
		if v < yMin {
			yMin = v
		}
		if v > yMax {
			yMax = v
		}
	}
	if yMax == yMin {
		yMax = yMin + 1
	}

	xScale := func(v float64) float64 {
		return marginLeft + (v-xMin)/(xMax-xMin)*plotW
	}
	yScale := func(v float64) float64 {
		return marginTop + plotH - (v-yMin)/(yMax-yMin)*plotH
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">`, width, height))
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="white"/>`, width, height))

	// Title
	sb.WriteString(fmt.Sprintf(
		`<text x="%d" y="%d" text-anchor="middle" font-size="16" font-family="sans-serif">Averaged Profile%s</text>`,
		width/2, 25, titleSuffix,
	))

	// Axis lines
	sb.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="black" stroke-width="1"/>`,
		marginLeft, marginTop, marginLeft, marginTop+plotH,
	))
	sb.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="black" stroke-width="1"/>`,
		marginLeft, marginTop+plotH, marginLeft+plotW, marginTop+plotH,
	))

	// Grid lines
	for k := 0; k <= 5; k++ {
		y := marginTop + float64(k)*plotH/5.0
		sb.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#ddd" stroke-width="1"/>`,
			marginLeft, y, marginLeft+plotW, y,
		))
	}

	// Data polyline
	var points []string
	for i := 0; i < n; i++ {
		points = append(points, fmt.Sprintf("%.1f,%.1f", xScale(distance[i]), yScale(data[i])))
	}
	sb.WriteString(fmt.Sprintf(
		`<polyline points="%s" fill="none" stroke="#2196F3" stroke-width="1.5"/>`,
		strings.Join(points, " "),
	))

	// X-axis labels
	numXLabels := 6
	for k := 0; k <= numXLabels; k++ {
		idx := int(float64(k) / float64(numXLabels) * float64(n-1))
		if idx >= n {
			idx = n - 1
		}
		val := distance[idx]
		x := xScale(val)
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="10" font-family="sans-serif">%.0f</text>`,
			x, marginTop+plotH+15, val,
		))
	}

	// Y-axis labels
	numYLabels := 5
	for k := 0; k <= numYLabels; k++ {
		frac := float64(k) / float64(numYLabels)
		val := yMin + frac*(yMax-yMin)
		y := yScale(val)
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="10" font-family="sans-serif">%.2e</text>`,
			marginLeft-5, y+3, val,
		))
	}

	// Axis titles
	sb.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-family="sans-serif" transform="rotate(-90,%.1f,%.1f)">Intensity</text>`,
		15.0, marginTop+plotH/2, 15.0, marginTop+plotH/2,
	))
	sb.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-family="sans-serif">Distance, m</text>`,
		marginLeft+plotW/2, marginTop+plotH+35,
	))

	sb.WriteString(`</svg>`)

	return &usecase.VisualizeResult{
		ContentType: "image/svg+xml",
		Body:        []byte(sb.String()),
	}, nil
}

// ========================= PNG generation =========================

func (u *visualizePreparedExperimentUseCaseImpl) heatmapToPNG(
	timeLabels []string,
	distanceLabels []string,
	zData [][]float64,
	titleSuffix string,
) (*usecase.VisualizeResult, error) {
	width, height := 900, 650
	marginLeft, marginRight, marginTop, marginBottom := 70.0, 55.0, 40.0, 80.0
	plotW := float64(width) - marginLeft - marginRight
	plotH := float64(height) - marginTop - marginBottom

	nTime := len(zData)          // columns → X = time
	nDist := len(distanceLabels) // rows → Y = distance
	if nTime == 0 || nDist == 0 {
		return nil, fmt.Errorf("empty data for heatmap")
	}

	// Transpose: zT[dist][time]
	zT := make([][]float64, nDist)
	for d := 0; d < nDist; d++ {
		row := make([]float64, nTime)
		for t := 0; t < nTime; t++ {
			if d < len(zData[t]) {
				row[t] = zData[t][d]
			}
		}
		zT[d] = row
	}

	cellW := plotW / float64(nTime)
	cellH := plotH / float64(nDist)

	// Percentile range for color scaling
	allVals := make([]float64, 0, nTime*nDist)
	for _, row := range zT {
		allVals = append(allVals, row...)
	}
	sort.Float64s(allVals)
	zMin := percentile(allVals, 0.05)
	zMax := percentile(allVals, 0.95)
	if zMax == zMin {
		zMax = zMin + 1
	}

	dc := gg.NewContext(width, height)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Load embedded Go font (no filesystem dependency)
	titleFont, err := loadFont(16)
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	dc.SetRGB(0, 0, 0)
	dc.SetFontFace(titleFont)
	dc.DrawStringAnchored("Lidar Heatmap"+titleSuffix, float64(width/2), 25, 0.5, 0.5)

	// Draw cells — flipped Y: d=0 at bottom
	for d := 0; d < nDist; d++ {
		for t := 0; t < nTime; t++ {
			val := zT[d][t]
			rr, g, b := heatmapColor((val - zMin) / (zMax - zMin))
			x := marginLeft + float64(t)*cellW
			y := marginTop + float64(nDist-1-d)*cellH
			dc.SetRGB(float64(rr)/255, float64(g)/255, float64(b)/255)
			dc.DrawRectangle(x, y, cellW+1, cellH+1)
			dc.Fill()
		}
	}

	// Grid lines (horizontal & vertical, dashed pattern via manual dash)
	dc.SetRGBA(0.85, 0.85, 0.85, 0.6)
	numHGrid := 8
	for k := 0; k <= numHGrid; k++ {
		y := marginTop + float64(k)*plotH/float64(numHGrid)
		drawDashedLineH(dc, marginLeft, marginLeft+plotW, y, 3, 3)
	}
	numVGrid := minInt(10, nTime-1)
	for k := 0; k <= numVGrid; k++ {
		x := marginLeft + float64(k)*plotW/float64(numVGrid)
		drawDashedLineV(dc, x, marginTop, marginTop+plotH, 3, 3)
	}

	// Axis lines
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	dc.DrawLine(marginLeft, marginTop, marginLeft, marginTop+plotH)
	dc.Stroke()
	dc.DrawLine(marginLeft, marginTop+plotH, marginLeft+plotW, marginTop+plotH)
	dc.Stroke()

	// Y-axis (distance) labels
	labelFont, _ := loadFont(10)
	dc.SetFontFace(labelFont)
	numYLabels := 8
	for k := 0; k <= numYLabels; k++ {
		idx := int(float64(k) / float64(numYLabels) * float64(nDist-1))
		if idx >= nDist {
			idx = nDist - 1
		}
		y := marginTop + float64(nDist-1-idx)*cellH + cellH/2
		dc.DrawStringAnchored(distanceLabels[idx]+" m", marginLeft-5, y+4, 1, 0.5)
	}

	// X-axis (time) labels
	smallFont, _ := loadFont(9)
	dc.SetFontFace(smallFont)
	numXLabels := minInt(10, nTime-1)
	for k := 0; k <= numXLabels; k++ {
		idx := int(float64(k) / float64(numXLabels) * float64(nTime-1))
		if idx >= nTime {
			idx = nTime - 1
		}
		x := marginLeft + float64(idx)*cellW + cellW/2
		dc.DrawStringAnchored(timeLabels[idx], x, marginTop+plotH+15, 0.5, 0)
	}

	// Axis titles
	axisFont, _ := loadFont(12)
	dc.SetFontFace(axisFont)
	dc.Push()
	dc.Translate(15, marginTop+plotH/2)
	dc.Rotate(-math.Pi / 2)
	dc.DrawStringAnchored("Distance, m", 0, 0, 0.5, 0.5)
	dc.Pop()
	dc.DrawStringAnchored("Time, HH:MM", marginLeft+plotW/2, marginTop+plotH+40, 0.5, 0)

	// Color bar
	barX, barY, barW, barH := marginLeft+plotW+10, marginTop, 15.0, plotH
	for k := 0; k < 100; k++ {
		t := float64(k) / 100.0
		rr, g, b := heatmapColor(t)
		y := barY + barH - (t * barH)
		dc.SetRGB(float64(rr)/255, float64(g)/255, float64(b)/255)
		dc.DrawRectangle(barX, y, barW, barH/100+1)
		dc.Fill()
	}
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	dc.DrawRectangle(barX, barY, barW, barH)
	dc.Stroke()

	// Colorbar ticks (≥5 evenly spaced)
	tickFont, _ := loadFont(10)
	dc.SetFontFace(tickFont)
	numTicks := 5
	tickLen := 6.0
	for k := 0; k <= numTicks; k++ {
		frac := 1.0 - float64(k)/float64(numTicks)
		tickY := barY + frac*barH
		val := zMin + (1.0-frac)*(zMax-zMin)
		dc.DrawLine(barX-tickLen, tickY, barX, tickY)
		dc.Stroke()
		dc.DrawStringAnchored(fmt.Sprintf("%.2e", val), barX-tickLen-3, tickY+4, 1, 0.5)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return &usecase.VisualizeResult{
		ContentType: "image/png",
		Body:        buf.Bytes(),
	}, nil
}

func (u *visualizePreparedExperimentUseCaseImpl) profileToPNG(
	distance []float64,
	data []float64,
	titleSuffix string,
) (*usecase.VisualizeResult, error) {
	width, height := 800, 500
	marginLeft, marginRight, marginTop, marginBottom := 70.0, 30.0, 40.0, 60.0
	plotW := float64(width) - marginLeft - marginRight
	plotH := float64(height) - marginTop - marginBottom

	n := len(data)
	if n == 0 {
		return nil, fmt.Errorf("empty profile data")
	}

	xMin, xMax := distance[0], distance[n-1]
	yMin, yMax := data[0], data[0]
	for _, v := range data {
		if v < yMin {
			yMin = v
		}
		if v > yMax {
			yMax = v
		}
	}
	if yMax == yMin {
		yMax = yMin + 1
	}

	xScale := func(v float64) float64 {
		return marginLeft + (v-xMin)/(xMax-xMin)*plotW
	}
	yScale := func(v float64) float64 {
		return marginTop + plotH - (v-yMin)/(yMax-yMin)*plotH
	}

	dc := gg.NewContext(width, height)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Load embedded Go font (no filesystem dependency)
	mainFont, err := loadFont(12)
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}

	// Title
	dc.SetRGB(0, 0, 0)
	dc.SetFontFace(mainFont)
	dc.DrawStringAnchored("Averaged Profile"+titleSuffix, float64(width/2), 25, 0.5, 0.5)

	// Axis lines
	dc.SetLineWidth(1)
	dc.DrawLine(marginLeft, marginTop, marginLeft, marginTop+plotH)
	dc.Stroke()
	dc.DrawLine(marginLeft, marginTop+plotH, marginLeft+plotW, marginTop+plotH)
	dc.Stroke()

	// Grid lines
	dc.SetRGBA(0.85, 0.85, 0.85, 0.6)
	for k := 0; k <= 5; k++ {
		y := marginTop + float64(k)*plotH/5.0
		dc.DrawLine(marginLeft, y, marginLeft+plotW, y)
		dc.Stroke()
	}

	// Data polyline
	dc.SetRGB(0.13, 0.59, 0.95) // #2196F3
	dc.SetLineWidth(1.5)
	dc.MoveTo(xScale(distance[0]), yScale(data[0]))
	for i := 1; i < n; i++ {
		dc.LineTo(xScale(distance[i]), yScale(data[i]))
	}
	dc.Stroke()

	// X-axis labels
	dc.SetFontFace(mainFont)
	numXLabels := 6
	for k := 0; k <= numXLabels; k++ {
		idx := int(float64(k) / float64(numXLabels) * float64(n-1))
		if idx >= n {
			idx = n - 1
		}
		val := distance[idx]
		x := xScale(val)
		dc.DrawStringAnchored(fmt.Sprintf("%.0f", val), x, marginTop+plotH+15, 0.5, 0)
	}

	// Y-axis labels
	numYLabels := 5
	for k := 0; k <= numYLabels; k++ {
		frac := float64(k) / float64(numYLabels)
		val := yMin + frac*(yMax-yMin)
		y := yScale(val)
		dc.DrawStringAnchored(fmt.Sprintf("%.2e", val), marginLeft-5, y+3, 1, 0.5)
	}

	// Axis titles
	dc.Push()
	dc.Translate(15, marginTop+plotH/2)
	dc.Rotate(-math.Pi / 2)
	dc.DrawStringAnchored("Intensity", 0, 0, 0.5, 0.5)
	dc.Pop()
	dc.DrawStringAnchored("Distance, m", marginLeft+plotW/2, marginTop+plotH+35, 0.5, 0)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return &usecase.VisualizeResult{
		ContentType: "image/png",
		Body:        buf.Bytes(),
	}, nil
}

// drawDashedLineH draws a horizontal dashed line made of individual segments.
func drawDashedLineH(dc *gg.Context, x1, x2, y, dashLen, gapLen float64) {
	segLen := dashLen + gapLen
	for x := x1; x < x2; x += segLen {
		endX := x + dashLen
		if endX > x2 {
			endX = x2
		}
		dc.DrawLine(x, y, endX, y)
		dc.Stroke()
	}
}

// drawDashedLineV draws a vertical dashed line made of individual segments.
func drawDashedLineV(dc *gg.Context, x, y1, y2, dashLen, gapLen float64) {
	segLen := dashLen + gapLen
	for y := y1; y < y2; y += segLen {
		endY := y + dashLen
		if endY > y2 {
			endY = y2
		}
		dc.DrawLine(x, y, x, endY)
		dc.Stroke()
	}
}

// loadFont loads the embedded Go font at the given size (no filesystem dependency).
func loadFont(size float64) (font.Face, error) {
	parsed, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse embedded font: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: size, DPI: 72})
	if err != nil {
		return nil, fmt.Errorf("create font face: %w", err)
	}
	return face, nil
}

// ========================= Plotly JSON generation =========================

type plotlyLayout struct {
	Title  string         `json:"title"`
	XAxis  plotlyAxis     `json:"xaxis"`
	YAxis  plotlyAxis     `json:"yaxis"`
	Margin plotlyMargin   `json:"margin"`
	Legend map[string]any `json:"legend,omitempty"`
}

type plotlyAxis struct {
	Title string `json:"title"`
}

type plotlyMargin struct {
	L int `json:"l"`
	R int `json:"r"`
	T int `json:"t"`
	B int `json:"b"`
}

type plotlyTrace struct {
	Type          string         `json:"type"`
	X             any            `json:"x"`
	Y             any            `json:"y"`
	Z             any            `json:"z,omitempty"`
	Colorscale    string         `json:"colorscale,omitempty"`
	Name          string         `json:"name,omitempty"`
	Hovertemplate string         `json:"hovertemplate,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	Line          map[string]any `json:"line,omitempty"`
}

type plotlyResponse struct {
	Data   []plotlyTrace `json:"data"`
	Layout plotlyLayout  `json:"layout"`
}

func (u *visualizePreparedExperimentUseCaseImpl) heatmapToPlotly(
	timeLabels []string,
	distanceLabels []string,
	zData [][]float64,
	titleSuffix string,
) (*usecase.VisualizeResult, error) {
	// zData is [time][distance], transpose to [distance][time] for Plotly heatmap (Y=dist, X=time)
	nTime := len(zData)
	nDist := len(distanceLabels)
	zT := make([][]float64, nDist)
	for d := 0; d < nDist; d++ {
		row := make([]float64, nTime)
		for t := 0; t < nTime; t++ {
			if d < len(zData[t]) {
				row[t] = zData[t][d]
			}
		}
		zT[d] = row
	}

	resp := plotlyResponse{
		Data: []plotlyTrace{
			{
				Type:          "heatmap",
				X:             timeLabels,
				Y:             distanceLabels,
				Z:             zT,
				Colorscale:    "Jet",
				Hovertemplate: "Time: %{x}<br>Distance: %{y} m<br>Intensity: %{z}<extra></extra>",
			},
		},
		Layout: plotlyLayout{
			Title:  "Lidar Heatmap" + titleSuffix,
			XAxis:  plotlyAxis{Title: "Time, HH:MM"},
			YAxis:  plotlyAxis{Title: "Distance, m"},
			Margin: plotlyMargin{L: 70, R: 30, T: 50, B: 80},
		},
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal plotly json: %w", err)
	}

	return &usecase.VisualizeResult{
		ContentType: "application/json",
		Body:        body,
	}, nil
}

func (u *visualizePreparedExperimentUseCaseImpl) profileToPlotly(
	distance []float64,
	data []float64,
	titleSuffix string,
) (*usecase.VisualizeResult, error) {
	resp := plotlyResponse{
		Data: []plotlyTrace{
			{
				Type:          "scatter",
				Mode:          "lines",
				X:             distance,
				Y:             data,
				Name:          "Averaged profile",
				Line:          map[string]any{"color": "#2196F3", "width": 2},
				Hovertemplate: "Distance: %{x:.0f} m<br>Intensity: %{y:.3e}<extra></extra>",
			},
		},
		Layout: plotlyLayout{
			Title:  "Averaged Profile" + titleSuffix,
			XAxis:  plotlyAxis{Title: "Distance, m"},
			YAxis:  plotlyAxis{Title: "Intensity"},
			Margin: plotlyMargin{L: 70, R: 30, T: 50, B: 60},
		},
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal plotly json: %w", err)
	}

	return &usecase.VisualizeResult{
		ContentType: "application/json",
		Body:        body,
	}, nil
}
