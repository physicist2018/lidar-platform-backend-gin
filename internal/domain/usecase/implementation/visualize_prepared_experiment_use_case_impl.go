package implementation

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/physicist2018/licelfile/v2/licelformat"
	"github.com/sirupsen/logrus"

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
	default:
		return u.profileToSVG(distance, avgData, titleSuffix)
	}
}

// ========================= SVG generation =========================

// formatTimeHHMM formats a Unix timestamp as HH:MM.
func formatTimeHHMM(unix int64) string {
	t := time.Unix(unix, 0).UTC()
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

	// Find min/max for color scaling
	zMin, zMax := zT[0][0], zT[0][0]
	for _, row := range zT {
		for _, v := range row {
			if v < zMin {
				zMin = v
			}
			if v > zMax {
				zMax = v
			}
		}
	}
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
	sb.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" font-size="9" font-family="sans-serif">%.2e</text>`,
		barX+barW+3, barY+barH, zMin,
	))
	sb.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" font-size="9" font-family="sans-serif">%.2e</text>`,
		barX+barW+3, barY+9, zMax,
	))

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

// applyFormula transforms profile data in-place according to the formula:
// "raw" — P (no change), "rangecorr" — P × r², "lograngecorr" — log₁₀(P × r²).
func applyFormula(data []float64, formula string, binWidth float64) {
	switch formula {
	case "rangecorr":
		for i, v := range data {
			r := float64(i) * binWidth / 1000.0
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
