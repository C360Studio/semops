package mavlink

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestGeneratedCommandLongUsesCanonicalWireOrderAndParses(t *testing.T) {
	generator := NewGenerator(255, 1)
	params := [7]float32{1, 2, 3, 4, 5, 6, 7}
	frame, err := generator.GenerateCommandLong(CommandLongMessage{
		Command:           CommandComponentArmDisarm,
		TargetSystemID:    42,
		TargetComponentID: 7,
		Confirmation:      1,
		Params:            params,
	})
	if err != nil {
		t.Fatalf("generate command long: %v", err)
	}

	if frame[0] != STXV2 {
		t.Fatalf("stx = 0x%02x, want 0x%02x", frame[0], STXV2)
	}
	if frame[1] != 33 {
		t.Fatalf("payload length = %d, want 33", frame[1])
	}
	payload := frame[HeaderSizeV2 : HeaderSizeV2+33]
	if got := math.Float32frombits(binary.LittleEndian.Uint32(payload[0:4])); got != params[0] {
		t.Fatalf("param1 wire field = %f, want %f", got, params[0])
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(payload[24:28])); got != params[6] {
		t.Fatalf("param7 wire field = %f, want %f", got, params[6])
	}
	if got := binary.LittleEndian.Uint16(payload[28:30]); got != CommandComponentArmDisarm {
		t.Fatalf("command wire field = %d, want %d", got, CommandComponentArmDisarm)
	}
	if payload[30] != 42 || payload[31] != 7 || payload[32] != 1 {
		t.Fatalf("target/confirmation = %d/%d/%d, want 42/7/1", payload[30], payload[31], payload[32])
	}

	packet := parseOne(t, frame)
	if packet.MessageID != MessageIDCommandLong {
		t.Fatalf("message id = %d, want %d", packet.MessageID, MessageIDCommandLong)
	}
	requireFloatField(t, packet, "param1", params[0])
	requireFloatField(t, packet, "param7", params[6])
	requireField[uint16](t, packet, "command", CommandComponentArmDisarm)
	requireField[uint8](t, packet, "target_system", 42)
	requireField[uint8](t, packet, "target_component", 7)
	requireField[uint8](t, packet, "confirmation", 1)
}

func TestGeneratedCommandAckParsesResult(t *testing.T) {
	generator := NewGenerator(42, 7)
	frame, err := generator.GenerateCommandAck(CommandAckMessage{
		Command:           CommandComponentArmDisarm,
		Result:            MAVResultAccepted,
		Progress:          100,
		ResultParam2:      -7,
		TargetSystemID:    255,
		TargetComponentID: 1,
	})
	if err != nil {
		t.Fatalf("generate command ack: %v", err)
	}

	if frame[1] != 10 {
		t.Fatalf("payload length = %d, want 10", frame[1])
	}
	packet := parseOne(t, frame)
	if packet.MessageID != MessageIDCommandAck {
		t.Fatalf("message id = %d, want %d", packet.MessageID, MessageIDCommandAck)
	}
	requireField[uint16](t, packet, "command", CommandComponentArmDisarm)
	requireField[uint8](t, packet, "result", MAVResultAccepted)
	requireField[uint8](t, packet, "progress", 100)
	requireField[int32](t, packet, "result_param2", -7)
	requireField[uint8](t, packet, "target_system", 255)
	requireField[uint8](t, packet, "target_component", 1)
}

func TestGeneratedRequestMessageCommandIsReadSideMVPShape(t *testing.T) {
	generator := NewGenerator(255, 190)
	frame, err := generator.GenerateCommandLong(CommandLongMessage{
		Command:           CommandRequestMessage,
		TargetSystemID:    1,
		TargetComponentID: 1,
		Params: [7]float32{
			float32(MessageIDAutopilotVersion),
		},
	})
	if err != nil {
		t.Fatalf("generate request message command: %v", err)
	}

	packet := parseOne(t, frame)
	requireField[uint16](t, packet, "command", CommandRequestMessage)
	requireFloatField(t, packet, "param1", float32(MessageIDAutopilotVersion))
	requireField[uint8](t, packet, "target_system", 1)
	requireField[uint8](t, packet, "target_component", 1)
}

func TestMAVResultString(t *testing.T) {
	tests := []struct {
		result uint8
		want   string
	}{
		{MAVResultAccepted, "accepted"},
		{MAVResultTemporarilyRejected, "temporarily_rejected"},
		{MAVResultDenied, "denied"},
		{MAVResultUnsupported, "unsupported"},
		{MAVResultFailed, "failed"},
		{MAVResultInProgress, "in_progress"},
		{99, "unknown(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := MAVResultString(tt.result); got != tt.want {
				t.Fatalf("result string = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestArduCopterModeMapping(t *testing.T) {
	tests := []struct {
		mode string
		want uint32
	}{
		{"STABILIZE", 0},
		{"guided", 4},
		{" LAND ", 9},
		{"SMART_RTL", 21},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got, err := ArduCopterCustomMode(tt.mode)
			if err != nil {
				t.Fatalf("custom mode: %v", err)
			}
			if got != tt.want {
				t.Fatalf("custom mode = %d, want %d", got, tt.want)
			}
			if name := ArduCopterModeName(got); name == "" {
				t.Fatalf("mode %d returned empty name", got)
			}
		})
	}

	if _, err := ArduCopterCustomMode("NOPE"); err == nil {
		t.Fatal("expected unknown mode error")
	}
	if got := ArduCopterModeName(999); got != "UNKNOWN(999)" {
		t.Fatalf("unknown mode name = %q", got)
	}
}
