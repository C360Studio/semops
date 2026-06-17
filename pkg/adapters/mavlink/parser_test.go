package mavlink

import (
	"encoding/binary"
	"math"
	"sync"
	"testing"
	"time"
)

func TestGeneratedHeartbeatParsesFields(t *testing.T) {
	generator := NewGenerator(42, 200)
	frame, err := generator.GenerateHeartbeat(HeartbeatMessage{
		VehicleType:    TypeQuadrotor,
		Autopilot:      AutopilotArduPilotMega,
		BaseMode:       ModeFlagStabilizeEnabled | ModeFlagManualInput,
		CustomMode:     7,
		SystemStatus:   StateStandby,
		MavlinkVersion: Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}

	packet := parseOne(t, frame)
	if packet.Version != Version2 {
		t.Fatalf("version = %d, want %d", packet.Version, Version2)
	}
	if packet.SystemID != 42 || packet.ComponentID != 200 {
		t.Fatalf("system/component = %d/%d, want 42/200", packet.SystemID, packet.ComponentID)
	}
	if packet.MessageID != MessageIDHeartbeat {
		t.Fatalf("message id = %d, want %d", packet.MessageID, MessageIDHeartbeat)
	}

	requireField[uint32](t, packet, "custom_mode", 7)
	requireField[uint8](t, packet, "type", TypeQuadrotor)
	requireField[uint8](t, packet, "autopilot", AutopilotArduPilotMega)
	requireField[uint8](t, packet, "system_status", StateStandby)
}

func TestGeneratedGlobalPositionParsesFields(t *testing.T) {
	generator := NewGenerator(1, 1)
	want := PositionMessage{
		TimeBootMs:  12345,
		Lat:         389000001,
		Lon:         -770000002,
		Alt:         120000,
		RelativeAlt: 45000,
		Vx:          321,
		Vy:          -12,
		Vz:          7,
		Hdg:         27000,
	}
	frame, err := generator.GenerateGlobalPosition(want)
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}

	packet := parseOne(t, frame)
	if packet.MessageID != MessageIDGlobalPositionInt {
		t.Fatalf("message id = %d, want %d", packet.MessageID, MessageIDGlobalPositionInt)
	}
	requireField[uint32](t, packet, "time_boot_ms", want.TimeBootMs)
	requireField[int32](t, packet, "lat", want.Lat)
	requireField[int32](t, packet, "lon", want.Lon)
	requireField[int32](t, packet, "alt", want.Alt)
	requireField[int16](t, packet, "vx", want.Vx)
	requireField[uint16](t, packet, "hdg", want.Hdg)
}

func TestGeneratedAttitudeParsesFields(t *testing.T) {
	generator := NewGenerator(1, 1)
	want := AttitudeMessage{
		TimeBootMs: 9876,
		Roll:       0.11,
		Pitch:      -0.05,
		Yaw:        1.57,
		Rollspeed:  0.01,
		Pitchspeed: 0.02,
		Yawspeed:   0.03,
	}
	frame, err := generator.GenerateAttitude(want)
	if err != nil {
		t.Fatalf("generate attitude: %v", err)
	}

	packet := parseOne(t, frame)
	if packet.MessageID != MessageIDAttitude {
		t.Fatalf("message id = %d, want %d", packet.MessageID, MessageIDAttitude)
	}
	requireField[uint32](t, packet, "time_boot_ms", want.TimeBootMs)
	requireFloatField(t, packet, "roll", want.Roll)
	requireFloatField(t, packet, "pitch", want.Pitch)
	requireFloatField(t, packet, "yaw", want.Yaw)
}

func TestGeneratedBatteryStatusUsesCanonicalWireOrder(t *testing.T) {
	generator := NewGenerator(1, 1)
	want := BatteryMessage{
		BatteryID:        3,
		BatteryFunction:  0,
		BatteryType:      1,
		Temperature:      2500,
		CurrentBattery:   -1500,
		CurrentConsumed:  1000,
		EnergyConsumed:   144000,
		BatteryRemaining: 85,
	}
	for i := 0; i < 4; i++ {
		want.Voltages[i] = 3700 + uint16(i)
	}
	frame, err := generator.GenerateBatteryStatus(want)
	if err != nil {
		t.Fatalf("generate battery: %v", err)
	}

	if frame[0] != STXV2 {
		t.Fatalf("stx = 0x%02x, want 0x%02x", frame[0], STXV2)
	}
	if frame[1] != 36 {
		t.Fatalf("payload length = %d, want 36", frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[HeaderSizeV2 : HeaderSizeV2+4]); got != uint32(want.CurrentConsumed) {
		t.Fatalf("current_consumed wire field = %d, want %d", got, want.CurrentConsumed)
	}
	if got := frame[HeaderSizeV2+32]; got != want.BatteryID {
		t.Fatalf("battery id wire field = %d, want %d", got, want.BatteryID)
	}
	if got := int8(frame[HeaderSizeV2+35]); got != want.BatteryRemaining {
		t.Fatalf("battery remaining wire field = %d, want %d", got, want.BatteryRemaining)
	}

	packet := parseOne(t, frame)
	requireField[int32](t, packet, "current_consumed", want.CurrentConsumed)
	requireField[int32](t, packet, "energy_consumed", want.EnergyConsumed)
	requireField[int16](t, packet, "current_battery", want.CurrentBattery)
	requireField[uint8](t, packet, "id", want.BatteryID)
	requireField[int8](t, packet, "battery_remaining", want.BatteryRemaining)

	voltages := requireFieldValue[[]uint16](t, packet, "voltages")
	for i := 0; i < 4; i++ {
		if voltages[i] != want.Voltages[i] {
			t.Fatalf("voltage[%d] = %d, want %d", i, voltages[i], want.Voltages[i])
		}
	}
}

func TestParserHandlesSplitBuffers(t *testing.T) {
	generator := NewGenerator(1, 1)
	frame, err := generator.GenerateGlobalPosition(PositionMessage{TimeBootMs: 100, Lat: 1, Lon: 2})
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}

	parser := NewParser()
	packets, err := parser.Parse(frame[:12])
	if err != nil {
		t.Fatalf("parse first chunk: %v", err)
	}
	if len(packets) != 0 {
		t.Fatalf("first chunk produced %d packets, want 0", len(packets))
	}

	packets, err = parser.Parse(frame[12:])
	if err != nil {
		t.Fatalf("parse second chunk: %v", err)
	}
	if len(packets) != 1 {
		t.Fatalf("second chunk produced %d packets, want 1", len(packets))
	}
	requireField[uint32](t, packets[0], "time_boot_ms", uint32(100))
}

func TestParserResyncsAcrossNoiseAndMultipleFrames(t *testing.T) {
	generator := NewGenerator(1, 1)
	heartbeat, err := generator.GenerateHeartbeat(HeartbeatMessage{
		VehicleType:    TypeQuadrotor,
		Autopilot:      AutopilotPX4,
		BaseMode:       ModeFlagSafetyArmed,
		SystemStatus:   StateActive,
		MavlinkVersion: Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	attitude, err := generator.GenerateAttitude(AttitudeMessage{TimeBootMs: 1, Roll: 0.1})
	if err != nil {
		t.Fatalf("generate attitude: %v", err)
	}

	parser := NewParser()
	stream := append([]byte{0x00, 0x01, 0x02}, heartbeat...)
	stream = append(stream, attitude...)
	packets, err := parser.Parse(stream)
	if err != nil {
		t.Fatalf("parse stream: %v", err)
	}
	if len(packets) != 2 {
		t.Fatalf("packets = %d, want 2", len(packets))
	}
	if packets[0].MessageID != MessageIDHeartbeat || packets[1].MessageID != MessageIDAttitude {
		t.Fatalf("message order = %d, %d", packets[0].MessageID, packets[1].MessageID)
	}
	if got := parser.Stats().InvalidPackets; got != 3 {
		t.Fatalf("invalid byte count = %d, want 3", got)
	}
}

func TestParserRejectsBadChecksum(t *testing.T) {
	generator := NewGenerator(1, 1)
	frame, err := generator.GenerateHeartbeat(HeartbeatMessage{MavlinkVersion: Version2})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	frame[len(frame)-1] ^= 0xff

	parser := NewParser()
	packets, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("parse corrupt frame: %v", err)
	}
	if len(packets) != 0 {
		t.Fatalf("corrupt frame produced %d packets, want 0", len(packets))
	}
	stats := parser.Stats()
	if stats.ChecksumErrors != 1 {
		t.Fatalf("checksum errors = %d, want 1", stats.ChecksumErrors)
	}
	if stats.InvalidPackets == 0 {
		t.Fatal("invalid packet counter did not move")
	}
}

func TestGeneratorSequenceIsConcurrentSafe(t *testing.T) {
	generator := NewGenerator(1, 1)
	const workers = 8
	const perWorker = 16

	var wg sync.WaitGroup
	sequences := make(chan uint8, workers*perWorker)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				frame, err := generator.GenerateHeartbeat(HeartbeatMessage{MavlinkVersion: Version2})
				if err != nil {
					t.Errorf("generate heartbeat: %v", err)
					return
				}
				sequences <- frame[4]
			}
		}()
	}
	wg.Wait()
	close(sequences)

	seen := make(map[uint8]bool, workers*perWorker)
	for sequence := range sequences {
		if seen[sequence] {
			t.Fatalf("duplicate sequence %d", sequence)
		}
		seen[sequence] = true
	}
	if len(seen) != workers*perWorker {
		t.Fatalf("sequences = %d, want %d", len(seen), workers*perWorker)
	}
}

func TestQuadcopterScenarioProducesParseableFlightStep(t *testing.T) {
	scenario := NewQuadcopterScenario(7, 38.9, -77.0, 120)
	scenario.AdvanceTime(500 * time.Millisecond)

	frames := make([][]byte, 0, 4)
	for _, next := range []func() ([]byte, error){
		scenario.NextHeartbeat,
		scenario.NextPosition,
		scenario.NextAttitude,
		scenario.NextBatteryStatus,
	} {
		frame, err := next()
		if err != nil {
			t.Fatalf("generate scenario frame: %v", err)
		}
		frames = append(frames, frame)
	}

	parser := NewParser()
	for _, frame := range frames {
		packets, err := parser.Parse(frame)
		if err != nil {
			t.Fatalf("parse scenario frame: %v", err)
		}
		if len(packets) != 1 {
			t.Fatalf("scenario frame produced %d packets, want 1", len(packets))
		}
	}
	if parser.Stats().ValidPackets != 4 {
		t.Fatalf("valid packets = %d, want 4", parser.Stats().ValidPackets)
	}
}

func parseOne(t *testing.T, frame []byte) *Packet {
	t.Helper()
	parser := NewParser()
	packets, err := parser.Parse(frame)
	if err != nil {
		t.Fatalf("parse frame: %v", err)
	}
	if len(packets) != 1 {
		t.Fatalf("packets = %d, want 1; stats=%+v", len(packets), parser.Stats())
	}
	return packets[0]
}

func requireField[T comparable](t *testing.T, packet *Packet, name string, want T) {
	t.Helper()
	got := requireFieldValue[T](t, packet, name)
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func requireFieldValue[T any](t *testing.T, packet *Packet, name string) T {
	t.Helper()
	if packet.ParsedFields == nil {
		t.Fatalf("packet has no parsed fields")
	}
	value, ok := packet.ParsedFields[name]
	if !ok {
		t.Fatalf("field %q missing from %+v", name, packet.ParsedFields)
	}
	typed, ok := value.(T)
	if !ok {
		t.Fatalf("field %q type = %T, want %T", name, value, typed)
	}
	return typed
}

func requireFloatField(t *testing.T, packet *Packet, name string, want float32) {
	t.Helper()
	got := requireFieldValue[float32](t, packet, name)
	if math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("%s = %f, want %f", name, got, want)
	}
}
