package visualize

import (
	"bytes"
	"fmt"
	"image/png"
	"math"

	"github.com/fogleman/gg"
)

// ProfileToPNG generates a PNG profile line chart.
func ProfileToPNG(
	distance []float64,
	data []float64,
	titleSuffix string,
) (*Result, error) {
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
	mainFont, err := LoadFont(12)
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

	return &Result{
		ContentType: "image/png",
		Body:        buf.Bytes(),
	}, nil
}
