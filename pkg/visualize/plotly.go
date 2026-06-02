package visualize

import (
	"encoding/json"
	"fmt"
)

type plotlyLayout struct {
	Title  string         `json:"title"`
	XAxis  plotlyAxis     `json:"xaxis"`
	YAxis  plotlyAxis     `json:"yaxis"`
	Margin plotlyMargin   `json:"margin"`
	Legend map[string]any `json:"legend,omitempty"`
}

type plotlyAxis struct {
	Title string `json:"title"`
}

type plotlyMargin struct {
	L int `json:"l"`
	R int `json:"r"`
	T int `json:"t"`
	B int `json:"b"`
}

type plotlyTrace struct {
	Type          string         `json:"type"`
	X             any            `json:"x"`
	Y             any            `json:"y"`
	Z             any            `json:"z,omitempty"`
	Colorscale    string         `json:"colorscale,omitempty"`
	Name          string         `json:"name,omitempty"`
	Hovertemplate string         `json:"hovertemplate,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	Line          map[string]any `json:"line,omitempty"`
}

type plotlyResponse struct {
	Data   []plotlyTrace `json:"data"`
	Layout plotlyLayout  `json:"layout"`
}

// HeatmapToPlotly generates a Plotly JSON representation of a heatmap.
func HeatmapToPlotly(
	timeLabels []string,
	distanceLabels []string,
	zData [][]float64,
	titleSuffix string,
) (*Result, error) {
	// zData is [time][distance], transpose to [distance][time] for Plotly heatmap (Y=dist, X=time)
	nTime := len(zData)
	nDist := len(distanceLabels)
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

	resp := plotlyResponse{
		Data: []plotlyTrace{
			{
				Type:          "heatmap",
				X:             timeLabels,
				Y:             distanceLabels,
				Z:             zT,
				Colorscale:    "Jet",
				Hovertemplate: "Time: %{x}<br>Distance: %{y} m<br>Intensity: %{z}<extra></extra>",
			},
		},
		Layout: plotlyLayout{
			Title:  "Lidar Heatmap" + titleSuffix,
			XAxis:  plotlyAxis{Title: "Time, HH:MM"},
			YAxis:  plotlyAxis{Title: "Distance, m"},
			Margin: plotlyMargin{L: 70, R: 30, T: 50, B: 80},
		},
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal plotly json: %w", err)
	}

	return &Result{
		ContentType: "application/json",
		Body:        body,
	}, nil
}

// ProfileToPlotly generates a Plotly JSON representation of a profile.
func ProfileToPlotly(
	distance []float64,
	data []float64,
	titleSuffix string,
) (*Result, error) {
	resp := plotlyResponse{
		Data: []plotlyTrace{
			{
				Type:          "scatter",
				Mode:          "lines",
				X:             distance,
				Y:             data,
				Name:          "Averaged profile",
				Line:          map[string]any{"color": "#2196F3", "width": 2},
				Hovertemplate: "Distance: %{x:.0f} m<br>Intensity: %{y:.3e}<extra></extra>",
			},
		},
		Layout: plotlyLayout{
			Title:  "Averaged Profile" + titleSuffix,
			XAxis:  plotlyAxis{Title: "Distance, m"},
			YAxis:  plotlyAxis{Title: "Intensity"},
			Margin: plotlyMargin{L: 70, R: 30, T: 50, B: 60},
		},
	}

	body, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal plotly json: %w", err)
	}

	return &Result{
		ContentType: "application/json",
		Body:        body,
	}, nil
}
