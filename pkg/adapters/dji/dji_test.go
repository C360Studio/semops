package dji

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseTelemetryRecordPreservesDJIShape(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "dji", "telemetry-media.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	record, err := ParseTelemetryRecord(data)
	if err != nil {
		t.Fatalf("parse DJI telemetry fixture: %v", err)
	}

	if record.FixtureClass != SyntheticTelemetryFixtureClass ||
		record.Source.Provider != "dji-shaped" ||
		record.Source.SourceID != "dji://fixture/matrice-350/alpha" ||
		record.Source.AircraftModel != "Matrice 350 RTK" ||
		record.Source.ControllerID != "pilot-controller-alpha" {
		t.Fatalf("source identity = %+v", record)
	}
	if record.ObservedAt != time.Date(2026, 6, 22, 18, 10, 0, 0, time.UTC) {
		t.Fatalf("observed_at = %s", record.ObservedAt)
	}
	if record.Aircraft.Latitude != 38.89945 ||
		record.Aircraft.Longitude != -77.03182 ||
		record.Aircraft.AltitudeMSLM == nil ||
		*record.Aircraft.AltitudeMSLM != 142.4 ||
		record.Aircraft.AltitudeAGLM == nil ||
		*record.Aircraft.AltitudeAGLM != 86.2 ||
		record.Aircraft.HeadingDeg == nil ||
		*record.Aircraft.HeadingDeg != 214.7 ||
		record.Aircraft.GroundSpeedMPS == nil ||
		*record.Aircraft.GroundSpeedMPS != 15.4 ||
		record.Aircraft.VerticalSpeedMPS == nil ||
		*record.Aircraft.VerticalSpeedMPS != -0.3 {
		t.Fatalf("aircraft state = %+v", record.Aircraft)
	}
	if record.Battery.Percent == nil ||
		*record.Battery.Percent != 68 ||
		record.Battery.VoltageV == nil ||
		*record.Battery.VoltageV != 48.2 {
		t.Fatalf("battery = %+v", record.Battery)
	}
	if record.Gimbal.YawDeg == nil ||
		*record.Gimbal.YawDeg != 91.4 ||
		record.Gimbal.PitchDeg == nil ||
		*record.Gimbal.PitchDeg != -24.3 ||
		record.Gimbal.Mode != "free" {
		t.Fatalf("gimbal = %+v", record.Gimbal)
	}
	if record.Camera.Payload != "Zenmuse H20T" ||
		record.Camera.Mode != "video" ||
		!record.Camera.Recording ||
		record.Camera.ZoomRatio == nil ||
		*record.Camera.ZoomRatio != 4 ||
		!record.Camera.ThermalEnabled {
		t.Fatalf("camera = %+v", record.Camera)
	}
	if len(record.MediaRefs) != 2 {
		t.Fatalf("media refs = %+v", record.MediaRefs)
	}
	firstMedia := record.MediaRefs[0]
	if firstMedia.URI != "object://semops/dji/matrice-alpha/clip-0001.mp4" ||
		firstMedia.Kind != "video/mp4" ||
		firstMedia.Role != "recorded-video" ||
		firstMedia.StartedAt == nil ||
		*firstMedia.StartedAt != time.Date(2026, 6, 22, 18, 9, 30, 0, time.UTC) ||
		firstMedia.EndedAt == nil ||
		*firstMedia.EndedAt != time.Date(2026, 6, 22, 18, 10, 0, 0, time.UTC) ||
		firstMedia.ByteLength == nil ||
		*firstMedia.ByteLength != 1048576 {
		t.Fatalf("first media ref = %+v", firstMedia)
	}
	if record.MediaRefs[1].Kind != "video/rtsp" ||
		record.MediaRefs[1].Role != "live-preview" {
		t.Fatalf("second media ref = %+v", record.MediaRefs[1])
	}
	if record.CommandAuthority.Mode != "local_operator" ||
		record.CommandAuthority.Holder != "pilot-alpha" ||
		record.CommandAuthority.RemoteCommandsEnabled ||
		!record.CommandAuthority.LocalOverrideRequired {
		t.Fatalf("command authority = %+v", record.CommandAuthority)
	}
}

func TestParseTelemetryRecordRejectsInvalidDJIShape(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing fixture class",
			body: `{"observed_at":"2026-06-22T18:10:00Z","source":{"provider":"dji-shaped","source_id":"dji://fixture/a"},"aircraft":{"latitude":38.9,"longitude":-77.0},"command_authority":{"mode":"local_operator"}}`,
			want: "fixture_class",
		},
		{
			name: "invalid latitude",
			body: `{"fixture_class":"semops.synthetic.dji.telemetry.v1","observed_at":"2026-06-22T18:10:00Z","source":{"provider":"dji-shaped","source_id":"dji://fixture/a"},"aircraft":{"latitude":138.9,"longitude":-77.0},"command_authority":{"mode":"local_operator"}}`,
			want: "latitude",
		},
		{
			name: "invalid battery percent",
			body: `{"fixture_class":"semops.synthetic.dji.telemetry.v1","observed_at":"2026-06-22T18:10:00Z","source":{"provider":"dji-shaped","source_id":"dji://fixture/a"},"aircraft":{"latitude":38.9,"longitude":-77.0},"battery":{"percent":168},"command_authority":{"mode":"local_operator"}}`,
			want: "battery percent",
		},
		{
			name: "relative media uri",
			body: `{"fixture_class":"semops.synthetic.dji.telemetry.v1","observed_at":"2026-06-22T18:10:00Z","source":{"provider":"dji-shaped","source_id":"dji://fixture/a"},"aircraft":{"latitude":38.9,"longitude":-77.0},"media_refs":[{"uri":"clip.mp4","kind":"video/mp4"}],"command_authority":{"mode":"local_operator"}}`,
			want: "uri must be absolute",
		},
		{
			name: "missing command authority",
			body: `{"fixture_class":"semops.synthetic.dji.telemetry.v1","observed_at":"2026-06-22T18:10:00Z","source":{"provider":"dji-shaped","source_id":"dji://fixture/a"},"aircraft":{"latitude":38.9,"longitude":-77.0}}`,
			want: "command authority",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTelemetryRecord([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
