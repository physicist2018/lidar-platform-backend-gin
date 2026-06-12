package meteo

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"math"
)

// Float64Slice represents a []float64 stored as bytea in PostgreSQL.
// Each float64 is stored as 8 bytes in little-endian format.
type Float64Slice []float64

// Scan implements the sql.Scanner interface for reading from DB.
func (s *Float64Slice) Scan(src interface{}) error {
	if src == nil {
		*s = nil
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	default:
		return fmt.Errorf("Float64Slice.Scan: unexpected type %T", src)
	}

	if len(data) == 0 {
		*s = Float64Slice{}
		return nil
	}

	if len(data)%8 != 0 {
		return fmt.Errorf("Float64Slice.Scan: data length %d is not a multiple of 8", len(data))
	}

	n := len(data) / 8
	result := make(Float64Slice, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
		result[i] = math.Float64frombits(bits)
	}
	*s = result
	return nil
}

// Value implements the driver.Valuer interface for writing to DB.
func (s Float64Slice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	if len(s) == 0 {
		return []byte{}, nil
	}

	data := make([]byte, len(s)*8)
	for i, v := range s {
		binary.LittleEndian.PutUint64(data[i*8:(i+1)*8], math.Float64bits(v))
	}
	return data, nil
}
