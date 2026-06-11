package licel

import (
	"encoding/binary"
	"math"
)

// Float64sToBytes converts a float64 slice to a byte array (LittleEndian).
func Float64sToBytes(data []float64) []byte {
	if len(data) == 0 {
		return nil
	}
	buf := make([]byte, len(data)*8)
	for i, v := range data {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

// BytesToFloat64s converts a byte array back to a float64 slice.
func BytesToFloat64s(data []byte) []float64 {
	if len(data) == 0 || len(data)%8 != 0 {
		return nil
	}
	result := make([]float64, len(data)/8)
	for i := range result {
		result[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return result
}
