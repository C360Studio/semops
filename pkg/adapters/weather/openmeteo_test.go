package weather

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseOpenMeteoPointForecastPreservesTacticalVariables(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "weather", "open-meteo-point.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	forecast, err := ParseOpenMeteoPointForecast(data)
	if err != nil {
		t.Fatalf("parse forecast: %v", err)
	}

	if forecast.Provider != ProviderOpenMeteo ||
		forecast.QueryShape != QueryShapePosition ||
		forecast.Latitude != 38.9 ||
		forecast.Longitude != -77.04 ||
		forecast.ElevationM == nil ||
		*forecast.ElevationM != 18 {
		t.Fatalf("forecast identity = %+v", forecast)
	}
	if forecast.Timezone != "GMT" ||
		forecast.UTCOffsetSeconds != 0 ||
		forecast.Units["wind_speed_10m"] != "km/h" ||
		forecast.Units["surface_pressure"] != "hPa" {
		t.Fatalf("forecast metadata/units = %+v", forecast)
	}
	if len(forecast.Samples) != 2 {
		t.Fatalf("samples = %+v", forecast.Samples)
	}

	first := forecast.Samples[0]
	if first.Time != time.Date(2026, 6, 22, 15, 0, 0, 0, time.UTC) {
		t.Fatalf("first time = %s", first.Time)
	}
	if first.TemperatureC == nil ||
		*first.TemperatureC != 29.4 ||
		first.PrecipitationMM == nil ||
		*first.PrecipitationMM != 0 ||
		first.VisibilityM == nil ||
		*first.VisibilityM != 16000 ||
		first.SurfacePressureHPA == nil ||
		*first.SurfacePressureHPA != 1004.1 ||
		first.WindSpeed10MKPH == nil ||
		*first.WindSpeed10MKPH != 12.5 ||
		first.WindGusts10MKPH == nil ||
		*first.WindGusts10MKPH != 22.1 ||
		first.WindDirection10Deg == nil ||
		*first.WindDirection10Deg != 210 ||
		first.WeatherCode == nil ||
		*first.WeatherCode != 3 {
		t.Fatalf("first tactical variables = %+v", first)
	}
	for _, field := range []string{
		"temperature_2m",
		"precipitation",
		"visibility",
		"surface_pressure",
		"wind_speed_10m",
		"wind_gusts_10m",
		"wind_direction_10m",
		"weather_code",
	} {
		if !hasString(first.SupportedFieldNames, field) {
			t.Fatalf("supported fields = %+v, missing %s", first.SupportedFieldNames, field)
		}
	}

	second := forecast.Samples[1]
	if second.Time != time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC) ||
		second.PrecipitationMM == nil ||
		*second.PrecipitationMM != 0.3 ||
		second.WeatherCode == nil ||
		*second.WeatherCode != 61 {
		t.Fatalf("second sample = %+v", second)
	}
}

func TestParseOpenMeteoPointForecastRejectsMalformedProviderShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing hourly",
			body: `{"latitude":38.9,"longitude":-77.04}`,
			want: "hourly",
		},
		{
			name: "missing time",
			body: `{"latitude":38.9,"longitude":-77.04,"hourly":{"temperature_2m":[20]}}`,
			want: "time",
		},
		{
			name: "mismatched variable length",
			body: `{"latitude":38.9,"longitude":-77.04,"hourly":{"time":["2026-06-22T15:00","2026-06-22T16:00"],"temperature_2m":[20]}}`,
			want: "temperature_2m length",
		},
		{
			name: "invalid coordinate",
			body: `{"latitude":138.9,"longitude":-77.04,"hourly":{"time":["2026-06-22T15:00"]}}`,
			want: "latitude",
		},
		{
			name: "non-integer weather code",
			body: `{"latitude":38.9,"longitude":-77.04,"hourly":{"time":["2026-06-22T15:00"],"weather_code":[3.4]}}`,
			want: "integer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseOpenMeteoPointForecast([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
