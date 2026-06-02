package visualize

import (
	"fmt"
	"math"
	"time"
)

// FormatTimeHHMM formats a unix timestamp as "HH:MM" in local time.
func FormatTimeHHMM(unix int64) string {
	t := time.Unix(unix, 0)
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

// MinInt returns the smaller of a and b.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Percentile returns the value at the given percentile (0..1) using linear interpolation.
// The input slice must be sorted in ascending order.
func Percentile(sorted []float64, p float64) float64 {
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

// ApplyFormula transforms profile data in-place according to the formula:
// "raw" — P (no change), "rangecorr" — P × r², "lograngecorr" — log₁₀(P × r²).
func ApplyFormula(data []float64, formula string, binWidth float64) {
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

// HeatmapColor returns RGB for a value in [0, 1] using a blue-to-red ramp.
func HeatmapColor(t float64) (r, g, b int) {
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
