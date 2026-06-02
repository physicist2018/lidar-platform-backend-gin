package visualize

import (
	"bytes"
	"fmt"
	"image/png"
	"math"
	"sort"

	"github.com/fogleman/gg"
)

// HeatmapToPNG generates a PNG heatmap image.
func HeatmapToPNG(
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

	// Percentile range for color scaling
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

	dc := gg.NewContext(width, height)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Load embedded Go font (no filesystem dependency)
	titleFont, err := LoadFont(16)
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
			rr, g, b := HeatmapColor((val - zMin) / (zMax - zMin))
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
		DrawDashedLineH(dc, marginLeft, marginLeft+plotW, y, 3, 3)
	}
	numVGrid := MinInt(10, nTime-1)
	for k := 0; k <= numVGrid; k++ {
		x := marginLeft + float64(k)*plotW/float64(numVGrid)
		DrawDashedLineV(dc, x, marginTop, marginTop+plotH, 3, 3)
	}

	// Axis lines
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	dc.DrawLine(marginLeft, marginTop, marginLeft, marginTop+plotH)
	dc.Stroke()
	dc.DrawLine(marginLeft, marginTop+plotH, marginLeft+plotW, marginTop+plotH)
	dc.Stroke()

	// Y-axis (distance) labels
	labelFont, _ := LoadFont(10)
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
	smallFont, _ := LoadFont(9)
	dc.SetFontFace(smallFont)
	numXLabels := MinInt(10, nTime-1)
	for k := 0; k <= numXLabels; k++ {
		idx := int(float64(k) / float64(numXLabels) * float64(nTime-1))
		if idx >= nTime {
			idx = nTime - 1
		}
		x := marginLeft + float64(idx)*cellW + cellW/2
		dc.DrawStringAnchored(timeLabels[idx], x, marginTop+plotH+15, 0.5, 0)
	}

	// Axis titles
	axisFont, _ := LoadFont(12)
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
		rr, g, b := HeatmapColor(t)
		y := barY + barH - (t * barH)
		dc.SetRGB(float64(rr)/255, float64(g)/255, float64(b)/255)
		dc.DrawRectangle(barX, y, barW, barH/100+1)
		dc.Fill()
	}
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(1)
	dc.DrawRectangle(barX, barY, barW, barH)
	dc.Stroke()

	// Colorbar ticks (≥5 evenly spaced, labels on the right)
	tickFont, _ := LoadFont(10)
	dc.SetFontFace(tickFont)
	numTicks := 5
	tickLen := 6.0
	for k := 0; k <= numTicks; k++ {
		frac := 1.0 - float64(k)/float64(numTicks)
		tickY := barY + frac*barH
		val := zMin + (1.0-frac)*(zMax-zMin)
		dc.DrawLine(barX+barW, tickY, barX+barW+tickLen, tickY)
		dc.Stroke()
		dc.DrawStringAnchored(fmt.Sprintf("%.2e", val), barX+barW+tickLen+3, tickY+4, 0, 0.5)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return &Result{
		ContentType: "image/png",
		Body:        buf.Bytes(),
	}, nil
}
