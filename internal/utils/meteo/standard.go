package meteo

import "math"

const (
	earthGravity   = 9.80665 // m/s²
	specificGasR   = 287.058 // J/(kg·K)
	lapseRate      = 0.0065  // K/m in troposphere
	seaLevelTempK  = 288.15  // K (15°C)
	seaLevelPres   = 1013.25 // hPa
	tropopauseAlt  = 11000.0 // m
	tropopauseTemp = 216.65  // K (-56.5°C)
	maxAlt         = 25000.0 // m
	step           = 100.0   // m
)

// StandardAtmosphere generates a standard atmosphere model (ISA).
// Returns altitudes from 0 to 25000 m with a step of 100 m.
// Only Pres, Hght, Temp are populated; optional fields are nil.
func StandardAtmosphere() *MeteoData {
	n := int(maxAlt/step) + 1
	data := &MeteoData{
		Pres: make([]float64, 0, n),
		Hght: make([]float64, 0, n),
		Temp: make([]float64, 0, n),
	}

	// Pressure at tropopause (used for stratosphere calculation)
	tropopausePres := seaLevelPres * math.Pow(tropopauseTemp/seaLevelTempK, earthGravity/(lapseRate*specificGasR))

	for h := 0.0; h <= maxAlt; h += step {
		data.Hght = append(data.Hght, h)

		if h <= tropopauseAlt {
			// Troposphere
			tK := seaLevelTempK - lapseRate*h
			tC := tK - 273.15
			p := seaLevelPres * math.Pow(tK/seaLevelTempK, earthGravity/(lapseRate*specificGasR))
			data.Temp = append(data.Temp, tC)
			data.Pres = append(data.Pres, p)
		} else {
			// Stratosphere — isothermal layer
			deltaH := h - tropopauseAlt
			tC := tropopauseTemp - 273.15
			p := tropopausePres * math.Exp(-(earthGravity*deltaH)/(specificGasR*tropopauseTemp))
			data.Temp = append(data.Temp, tC)
			data.Pres = append(data.Pres, p)
		}
	}

	return data
}
