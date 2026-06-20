package adsb

import (
	"strings"
	"testing"
	"time"
)

func TestParseOpenSkySnapshotPreservesNullableStateVectorFields(t *testing.T) {
	snapshot, err := ParseOpenSkySnapshot([]byte(sampleOpenSkySnapshot))
	if err != nil {
		t.Fatalf("parse opensky snapshot: %v", err)
	}

	if snapshot.Time != time.Date(2026, 6, 20, 14, 30, 0, 0, time.UTC) {
		t.Fatalf("snapshot time = %s", snapshot.Time)
	}
	if len(snapshot.States) != 4 {
		t.Fatalf("states = %+v", snapshot.States)
	}

	normal := snapshot.States[0]
	if normal.ICAO24 != "a1b2c3" ||
		normal.Callsign == nil ||
		*normal.Callsign != "N123AB" ||
		normal.OriginCountry != "United States" {
		t.Fatalf("normal state identity = %+v", normal)
	}
	if normal.TimePosition == nil ||
		*normal.TimePosition != time.Date(2026, 6, 20, 14, 29, 45, 0, time.UTC) ||
		normal.LastContact != time.Date(2026, 6, 20, 14, 29, 52, 0, time.UTC) {
		t.Fatalf("normal state times = %+v", normal)
	}
	if !normal.HasPosition() ||
		normal.Latitude == nil ||
		normal.Longitude == nil ||
		*normal.Latitude != 38.9 ||
		*normal.Longitude != -77.04 {
		t.Fatalf("normal position = lat=%v lon=%v", normal.Latitude, normal.Longitude)
	}
	if normal.PositionSource != PositionSourceADSB ||
		normal.PositionSourceLabel() != "ads-b" ||
		normal.Category == nil ||
		*normal.Category != 2 {
		t.Fatalf("normal source/category = %+v", normal)
	}
	if normal.VelocityMPS == nil ||
		*normal.VelocityMPS != 71.5 ||
		normal.TrueTrackDeg == nil ||
		*normal.TrueTrackDeg != 180.25 ||
		normal.VerticalRateMPS == nil ||
		*normal.VerticalRateMPS != -1.2 {
		t.Fatalf("normal motion = %+v", normal)
	}
	if len(normal.SensorIDs) != 2 || normal.SensorIDs[0] != 101 || normal.SensorIDs[1] != 202 {
		t.Fatalf("normal sensors = %+v", normal.SensorIDs)
	}

	missingPosition := snapshot.States[1]
	if missingPosition.Callsign != nil ||
		missingPosition.TimePosition != nil ||
		missingPosition.HasPosition() ||
		missingPosition.Latitude != nil ||
		missingPosition.Longitude != nil {
		t.Fatalf("missing-position state = %+v", missingPosition)
	}
	if missingPosition.PositionSourceLabel() != "ads-b" {
		t.Fatalf("missing-position source = %q", missingPosition.PositionSourceLabel())
	}

	mlat := snapshot.States[2]
	if mlat.PositionSource != PositionSourceMLAT ||
		mlat.PositionSourceLabel() != "mlat" ||
		mlat.Category == nil ||
		*mlat.Category != 14 {
		t.Fatalf("mlat state = %+v", mlat)
	}

	unknown := snapshot.States[3]
	if unknown.HasPositionSource ||
		unknown.PositionSourceLabel() != "unknown" {
		t.Fatalf("unknown-source state = %+v", unknown)
	}
}

func TestParseOpenSkySnapshotRejectsMalformedRows(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing snapshot time",
			body: `{"states":[]}`,
			want: "time",
		},
		{
			name: "missing states",
			body: `{"time":1781965800}`,
			want: "states",
		},
		{
			name: "short row",
			body: `{"time":1781965800,"states":[["a1b2c3"]]}`,
			want: "expected at least 17 fields",
		},
		{
			name: "missing icao",
			body: strings.Replace(sampleOpenSkySnapshot, `"a1b2c3"`, `null`, 1),
			want: "icao24",
		},
		{
			name: "invalid on_ground",
			body: strings.Replace(sampleOpenSkySnapshot, "false,\n      71.5", "\"not-bool\",\n      71.5", 1),
			want: "on_ground",
		},
		{
			name: "invalid optional longitude",
			body: strings.Replace(sampleOpenSkySnapshot, "-77.04", `"west"`, 1),
			want: "longitude",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseOpenSkySnapshot([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

const sampleOpenSkySnapshot = `{
  "time": 1781965800,
  "states": [
    [
      "a1b2c3",
      "N123AB  ",
      "United States",
      1781965785,
      1781965792,
      -77.04,
      38.9,
      1200.5,
      false,
      71.5,
      180.25,
      -1.2,
      [101, 202],
      1250.75,
      "1200",
      false,
      0,
      2
    ],
    [
      "d4e5f6",
      null,
      "Canada",
      null,
      1781965790,
      null,
      null,
      null,
      false,
      null,
      null,
      null,
      null,
      null,
      null,
      false,
      0,
      0
    ],
    [
      "abc123",
      "UAV42",
      "United Kingdom",
      1781965770,
      1781965795,
      -77.05,
      38.91,
      800,
      false,
      45.5,
      91,
      0,
      null,
      820,
      null,
      true,
      2,
      14
    ],
    [
      "ffff01",
      "GLIDER",
      "Germany",
      1781965765,
      1781965791,
      8.12,
      49.01,
      900,
      false,
      30,
      270,
      0,
      null,
      905,
      null,
      false,
      null,
      null
    ]
  ]
}`
