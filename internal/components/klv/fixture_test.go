package klv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMISB0601TruthFixtureEncodesAndDecodesDeterministically(t *testing.T) {
	truth := loadMISB0601TruthFixture(t)

	packet, err := truth.PacketPayload()
	if err != nil {
		t.Fatalf("build packet payload from truth fixture: %v", err)
	}
	if packet.Source != DefaultDemuxSource {
		t.Fatalf("packet source = %q, want %q", packet.Source, DefaultDemuxSource)
	}
	if packet.MediaRef != truth.MediaRef || packet.PacketRef != truth.PacketRef {
		t.Fatalf("packet refs = %q/%q, want %q/%q", packet.MediaRef, packet.PacketRef, truth.MediaRef, truth.PacketRef)
	}
	if len(packet.PacketBytes) == 0 {
		t.Fatal("encoded KLV packet bytes are empty")
	}

	frame, err := DecodeMISB0601Packet(packet)
	if err != nil {
		t.Fatalf("decode deterministic KLV truth fixture: %v", err)
	}
	if !frame.FrameTime.Equal(truth.FrameTime) {
		t.Fatalf("frame time = %s, want %s", frame.FrameTime, truth.FrameTime)
	}
	if frame.PlatformDesignation != truth.PlatformDesignation {
		t.Fatalf("platform designation = %q, want %q", frame.PlatformDesignation, truth.PlatformDesignation)
	}
	requireTruthClose(t, "sensor latitude", frame.SensorLatitude, truth.SensorLatitude, signed32QuantizationTolerance(90))
	requireTruthClose(t, "sensor longitude", frame.SensorLongitude, truth.SensorLongitude, signed32QuantizationTolerance(180))
	requireTruthClose(t, "sensor altitude", frame.SensorAltitudeMeters, truth.SensorAltitudeMeters, unsignedQuantizationTolerance(-900, 19000, misbMaxUint16))
	requireTruthClose(t, "sensor azimuth", frame.SensorAzimuthDegrees, truth.SensorAzimuthDegrees, unsignedQuantizationTolerance(0, 360, misbMaxUint32))
	requireTruthClose(t, "sensor elevation", frame.SensorElevationDegrees, truth.SensorElevationDegrees, signed32QuantizationTolerance(180))
	requireTruthClose(t, "frame center latitude", frame.FrameCenterLatitude, truth.FrameCenterLatitude, signed32QuantizationTolerance(90))
	requireTruthClose(t, "frame center longitude", frame.FrameCenterLongitude, truth.FrameCenterLongitude, signed32QuantizationTolerance(180))
	requireTruthClose(t, "frame center elevation", frame.FrameCenterElevationMeters, truth.FrameCenterElevationMeters, unsignedQuantizationTolerance(-900, 19000, misbMaxUint16))

	for _, field := range []string{
		"PrecisionTimeStamp",
		"PlatformDesignation",
		"SensorLatitude",
		"SensorLongitude",
		"SensorTrueAltitude",
		"SensorRelativeAzimuthAngle",
		"SensorRelativeElevationAngle",
		"FrameCenterLatitude",
		"FrameCenterLongitude",
		"FrameCenterElevation",
	} {
		requireField(t, frame.Fields, field)
	}
}

func TestEncodeMISB0601TruthRejectsOutOfRangeValues(t *testing.T) {
	truth := loadMISB0601TruthFixture(t)
	invalidLatitude := 91.0
	truth.SensorLatitude = &invalidLatitude

	_, err := EncodeMISB0601Truth(truth)
	if err == nil {
		t.Fatal("expected out-of-range sensor latitude to fail")
	}
	if !strings.Contains(err.Error(), "sensor_latitude") {
		t.Fatalf("error = %q, want sensor_latitude context", err)
	}
}

func loadMISB0601TruthFixture(t *testing.T) MISB0601Truth {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "fixtures", "klv", "misb0601-truth.json"))
	if err != nil {
		t.Fatalf("read deterministic KLV truth fixture: %v", err)
	}
	var truth MISB0601Truth
	if err := json.Unmarshal(data, &truth); err != nil {
		t.Fatalf("unmarshal deterministic KLV truth fixture: %v", err)
	}
	return truth
}

func requireTruthClose(t *testing.T, name string, got *float64, want *float64, tolerance float64) {
	t.Helper()
	if want == nil {
		t.Fatalf("%s truth is nil", name)
	}
	requireClose(t, name, got, *want, tolerance)
}

func signed32QuantizationTolerance(maxAbs float64) float64 {
	return maxAbs/misbMaxInt32 + 1e-12
}

func unsignedQuantizationTolerance(minValue, maxValue, maxRaw float64) float64 {
	return (maxValue-minValue)/maxRaw + 1e-12
}
