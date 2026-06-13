package controller

import (
	"strconv"
	"time"
)

const defaultTimeLayout = time.RFC3339

// parseUint parses a string as uint.
func parseUint(s string) (uint, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

// parseInt parses a string as int with a fallback default value.
func parseInt(s string, defaultVal int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseFloat64 parses a string as float64 with a fallback default value.
func parseFloat64(s string, defaultVal float64) float64 {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

// strDefault returns the string value or a default if empty.
func strDefault(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}

// parseTimeQuery parses a time string in RFC3339 format. Returns nil if the string is empty.
func parseTimeQuery(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(defaultTimeLayout, s)
	if err != nil {
		return nil
	}
	return &t
}
