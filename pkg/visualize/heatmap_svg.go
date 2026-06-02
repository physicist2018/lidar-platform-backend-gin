package visualize

import (
	"fmt"
	"sort"
	"strings"
)

// HeatmapToSVG generates an SVG heatmap image.
func HeatmapToSVG(
	timeLabels []string,
	distanceLabels []string,
	zData [][]float64,
	titleSuffix string,
) (*Result, error) {
	width, height := 950, 650
	marginLeft, marginRight, marginTop, marginBottom := 70.0, 100.0, 40.0, 80.0
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
	allVals := make([]float64, 0, nTime*nDist)
	for _, row := range zT {
		allVals = append(allVals, row...)
	}
	sort.Float64s(allVals)
	zMin := Percentile(allVals, 0.05)
	zMax := Percentile(allVals, 0.95)
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
			r, g, bb := HeatmapColor((val - zMin) / (zMax - zMin))
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
	numVGrid := MinInt(10, nTime-1)
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
	numXLabels := MinInt(10, nTime-1)
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
		r, g, bb := HeatmapColor(t)
		y := barY + barH - (t * barH)
		sb.WriteString(fmt.Sprintf(
			`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="rgb(%d,%d,%d)"/>`,
			barX, y, barW, barH/100+1, r, g, bb,
		))
	}
	sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="none" stroke="black"/>`, barX, barY, barW, barH))

	// Colorbar ticks (≥5 evenly spaced, labels on the right)
	numTicks := 5
	tickLen := 6.0
	for k := 0; k <= numTicks; k++ {
		frac := 1.0 - float64(k)/float64(numTicks) // 0=bottom, 1=top
		tickY := barY + frac*barH
		val := zMin + (1.0-frac)*(zMax-zMin)
		sb.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="black" stroke-width="1"/>`,
			barX+barW, tickY, barX+barW+tickLen, tickY,
		))
		sb.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="start" font-size="10" font-family="sans-serif">%.2e</text>`,
			barX+barW+tickLen+3, tickY+4, val,
		))
	}

	sb.WriteString(`</svg>`)

	return &Result{
		ContentType: "image/svg+xml",
		Body:        []byte(sb.String()),
	}, nil
}
