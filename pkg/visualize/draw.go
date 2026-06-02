package visualize

import (
	"fmt"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// DrawDashedLineH draws a horizontal dashed line made of individual segments.
func DrawDashedLineH(dc *gg.Context, x1, x2, y, dashLen, gapLen float64) {
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

// DrawDashedLineV draws a vertical dashed line made of individual segments.
func DrawDashedLineV(dc *gg.Context, x, y1, y2, dashLen, gapLen float64) {
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

// LoadFont loads the embedded Go font at the given size (no filesystem dependency).
func LoadFont(size float64) (font.Face, error) {
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
