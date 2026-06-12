package meteo

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MeteoData holds all parsed meteo columns as slices.
// Pres, Hght, Temp are always present (filled from file or standard atmosphere).
// Relh, Mixr, Drct, Sknt may be nil if data is unavailable.
type MeteoData struct {
	Pres []float64
	Hght []float64
	Temp []float64
	Relh []float64
	Mixr []float64
	Drct []float64
	Sknt []float64
}

// ParseMeteoFile parses a meteo.dat file into a MeteoData struct.
// Expected format (7 columns separated by whitespace):
//
//	PRES   HGHT   TEMP   DWPT   RELH   MIXR   DRCT   SKNT   THTA   THTE   THTV
//	hPa     m      C      C      %    g/kg    deg   knot     K      K      K
//	 1006.0     82  -14.1  -17.5     75   0.97      0      0  258.6  261.3  258.8
//	 ...
//
// Columns extracted: PRES, HGHT, TEMP, RELH, MIXR, DRCT, SKNT.
// PRES, HGHT, TEMP are required; if parsing fails, an error is returned.
// RELH, MIXR, DRCT, SKNT are optional; unparseable values are stored as 0,
// but if ALL values in a column are 0/missing, the slice is set to nil.
func ParseMeteoFile(path string) (*MeteoData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open meteo file: %w", err)
	}
	defer f.Close()

	data := &MeteoData{}
	scanner := bufio.NewScanner(f)
	lineNum := 0

	// Track optional column presence: if we never see a non-default value for an
	// optional column, keep it nil.
	hasRelh, hasMixr, hasDrct, hasSknt := false, false, false, false

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Skip header/separator lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "PRES") {
			continue
		}

		// Split on whitespace
		fields := strings.Fields(trimmed)
		if len(fields) < 3 {
			continue // not enough data for required fields
		}

		// Required columns (indices based on the header order):
		//   0: PRES, 1: HGHT, 2: TEMP, 3: DWPT(skip), 4: RELH, 5: MIXR, 6: DRCT, 7: SKNT
		pres, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: parse PRES %q: %w", lineNum, fields[0], err)
		}
		hght, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: parse HGHT %q: %w", lineNum, fields[1], err)
		}
		temp, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: parse TEMP %q: %w", lineNum, fields[2], err)
		}

		data.Pres = append(data.Pres, pres)
		data.Hght = append(data.Hght, hght)
		data.Temp = append(data.Temp, temp)

		// Optional fields — try to parse; if missing or unparseable, use 0
		data.Relh = appendOptionalFloat64(data.Relh, getField(fields, 4), &hasRelh)
		data.Mixr = appendOptionalFloat64(data.Mixr, getField(fields, 5), &hasMixr)
		data.Drct = appendOptionalFloat64(data.Drct, getField(fields, 6), &hasDrct)
		data.Sknt = appendOptionalFloat64(data.Sknt, getField(fields, 7), &hasSknt)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read meteo file: %w", err)
	}

	if len(data.Pres) == 0 {
		return nil, fmt.Errorf("meteo file contains no data rows")
	}

	// If optional columns were never populated with a non-zero value, set to nil
	if !hasRelh {
		data.Relh = nil
	}
	if !hasMixr {
		data.Mixr = nil
	}
	if !hasDrct {
		data.Drct = nil
	}
	if !hasSknt {
		data.Sknt = nil
	}

	return data, nil
}

func getField(fields []string, idx int) string {
	if idx >= len(fields) {
		return ""
	}
	return fields[idx]
}

func appendOptionalFloat64(slice []float64, field string, hasNonDefault *bool) []float64 {
	if field == "" {
		return append(slice, 0)
	}
	val, err := strconv.ParseFloat(field, 64)
	if err != nil {
		return append(slice, 0)
	}
	if val != 0 {
		*hasNonDefault = true
	}
	return append(slice, val)
}
