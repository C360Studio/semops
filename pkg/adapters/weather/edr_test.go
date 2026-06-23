package weather

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseOGCEDRPositionForecastPreservesTacticalVariables(t *testing.T) {
	data := readOGCEDRFixture(t)

	forecast, err := ParseOGCEDRPositionForecast(data)
	if err != nil {
		t.Fatalf("parse OGC EDR forecast: %v", err)
	}

	if forecast.Provider != ProviderOGCEDR ||
		forecast.QueryShape != QueryShapePosition ||
		forecast.Latitude != 38.9 ||
		forecast.Longitude != -77.04 ||
		forecast.ElevationM == nil ||
		*forecast.ElevationM != 18 {
		t.Fatalf("forecast identity = %+v", forecast)
	}
	if forecast.Units["temperature_2m"] != "Cel" ||
		forecast.Units["wind_speed_10m"] != "km/h" ||
		forecast.Units["surface_pressure"] != "hPa" {
		t.Fatalf("forecast units = %+v", forecast.Units)
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
		second.WindGusts10MKPH == nil ||
		*second.WindGusts10MKPH != 31.4 ||
		second.WeatherCode == nil ||
		*second.WeatherCode != 61 {
		t.Fatalf("second sample = %+v", second)
	}
}

func TestParseOGCEDRPositionForecastRejectsMalformedShapes(t *testing.T) {
	base := string(readOGCEDRFixture(t))
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "wrong query type",
			body: strings.Replace(base, `"query_type": "position"`, `"query_type": "area"`, 1),
			want: "query_type",
		},
		{
			name: "invalid point WKT",
			body: strings.Replace(base, `"coords": "POINT(-77.04 38.9)"`, `"coords": "POLYGON((-77 38,-76 38,-76 39,-77 39,-77 38))"`, 1),
			want: "WKT POINT",
		},
		{
			name: "mismatched range length",
			body: strings.Replace(base, `"values": [29.4, 28.9]`, `"values": [29.4]`, 1),
			want: "temperature_2m length",
		},
		{
			name: "non-integer weather code",
			body: strings.Replace(base, `"values": [3, 61]`, `"values": [3.4, 61]`, 1),
			want: "integer",
		},
		{
			name: "missing ranges",
			body: `{
				"fixture_class":"semops.synthetic.ogc-edr.position.v1",
				"edr":{"query_type":"position","query":{"coords":"POINT(-77.04 38.9)"}},
				"coverage":{"type":"Coverage","domain":{"axes":{"x":{"values":[-77.04]},"y":{"values":[38.9]},"t":{"values":["2026-06-22T15:00:00Z"]}}}}
			}`,
			want: "ranges",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseOGCEDRPositionForecast([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestParseOGCEDRSpatialForecastAcceptsAreaTrajectoryAndCorridor(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		shape           string
		geometry        string
		coordinateCount int
		wantGust        float64
		wantWidth       *float64
		wantHeight      *float64
	}{
		{
			name:            "area",
			path:            filepath.Join("..", "..", "..", "fixtures", "weather", "ogc-edr-area.json"),
			shape:           QueryShapeArea,
			geometry:        "POLYGON",
			coordinateCount: 5,
			wantGust:        41.2,
		},
		{
			name:            "trajectory",
			path:            filepath.Join("..", "..", "..", "fixtures", "weather", "ogc-edr-trajectory.json"),
			shape:           QueryShapeTrajectory,
			geometry:        "LINESTRING",
			coordinateCount: 3,
			wantGust:        36.1,
		},
		{
			name:            "corridor",
			path:            filepath.Join("..", "..", "..", "fixtures", "weather", "ogc-edr-corridor.json"),
			shape:           QueryShapeCorridor,
			geometry:        "LINESTRING",
			coordinateCount: 3,
			wantGust:        40,
			wantWidth:       floatPtr(500),
			wantHeight:      floatPtr(150),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			forecast, err := ParseOGCEDRSpatialForecast(data)
			if err != nil {
				t.Fatalf("parse spatial forecast: %v", err)
			}
			if forecast.Provider != ProviderOGCEDR ||
				forecast.QueryShape != tt.shape ||
				forecast.GeometryType != tt.geometry ||
				forecast.CoordinateCount != tt.coordinateCount ||
				forecast.CRS != "http://www.opengis.net/def/crs/OGC/1.3/CRS84" ||
				len(forecast.Samples) != 2 {
				t.Fatalf("forecast identity = %+v", forecast)
			}
			if forecast.Units["wind_gusts_10m"] != "km/h" {
				t.Fatalf("forecast units = %+v", forecast.Units)
			}
			second := forecast.Samples[1]
			if second.Time != time.Date(2026, 6, 22, 16, 0, 0, 0, time.UTC) ||
				second.WindGusts10MKPH == nil ||
				*second.WindGusts10MKPH != tt.wantGust ||
				second.WeatherCode == nil {
				t.Fatalf("second sample = %+v", second)
			}
			if !sameFloatPtr(forecast.CorridorWidth, tt.wantWidth) ||
				!sameFloatPtr(forecast.CorridorHeight, tt.wantHeight) {
				t.Fatalf("corridor dimensions = width %v height %v", forecast.CorridorWidth, forecast.CorridorHeight)
			}
		})
	}
}

func TestParseOGCEDRSpatialForecastRejectsMalformedShapes(t *testing.T) {
	area := string(readFixtureFile(t, "ogc-edr-area.json"))
	corridor := string(readFixtureFile(t, "ogc-edr-corridor.json"))
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "area requires polygon",
			body: strings.Replace(area,
				`"coords": "POLYGON((-77.08 38.88,-77.00 38.88,-77.00 38.94,-77.08 38.94,-77.08 38.88))"`,
				`"coords": "LINESTRING(-77.06 38.88,-77.04 38.90)"`,
				1,
			),
			want: "POLYGON",
		},
		{
			name: "area polygon ring must be closed",
			body: strings.Replace(area,
				`"coords": "POLYGON((-77.08 38.88,-77.00 38.88,-77.00 38.94,-77.08 38.94,-77.08 38.88))"`,
				`"coords": "POLYGON((-77.08 38.88,-77.00 38.88,-77.00 38.94,-77.08 38.94))"`,
				1,
			),
			want: "closed",
		},
		{
			name: "corridor requires width",
			body: strings.Replace(corridor, `"corridor-width": 500,`, "", 1),
			want: "corridor-width",
		},
		{
			name: "corridor requires line",
			body: strings.Replace(corridor,
				`"coords": "LINESTRING(-77.06 38.88,-77.04 38.90,-77.02 38.93)"`,
				`"coords": "POLYGON((-77.08 38.88,-77.00 38.88,-77.00 38.94,-77.08 38.94,-77.08 38.88))"`,
				1,
			),
			want: "LINESTRING",
		},
		{
			name: "position fixture is not spatial forecast",
			body: string(readOGCEDRFixture(t)),
			want: "spatial query_type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseOGCEDRSpatialForecast([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func readOGCEDRFixture(t *testing.T) []byte {
	t.Helper()
	return readFixtureFile(t, "ogc-edr-position.json")
}

func readFixtureFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "weather", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func floatPtr(value float64) *float64 {
	return &value
}

func sameFloatPtr(got *float64, want *float64) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}
