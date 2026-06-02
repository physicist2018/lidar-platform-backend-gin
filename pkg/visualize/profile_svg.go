package visualize

import (
	"fmt"
	"strings"
)

// ProfileToSVG generates an SVG profile line chart.
func ProfileToSVG(
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

	return &Result{
		ContentType: "image/svg+xml",
		Body:        []byte(sb.String()),
	}, nil
}
