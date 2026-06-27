package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

var errExpectedCommandLong = errors.New("expected COMMAND_LONG")

func TestRunDryRunBuildsReadSideMVPCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"-confirm-simulator-only",
		"-dry-run",
		"-route", "udp://127.0.0.1:14540",
		"-target-system", "1",
		"-target-component", "1",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run dry-run: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"action=request_autopilot_version",
		"command=512",
		"request_message=148",
		"source=255/190",
		"target=1/1",
		"expected_ack_task_suffix=system-1-command-512-target-255-190",
		"frame_hex=",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, output)
		}
	}
}

func TestBuildCommandFrameRejectsWithoutSimulatorConfirmation(t *testing.T) {
	_, _, err := buildCommandFrame(config{Action: actionRequestAutopilotVersion})
	if err == nil {
		t.Fatal("expected simulator confirmation error")
	}
	if !strings.Contains(err.Error(), "simulator-only confirmation is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildGCSHeartbeatFrameUsesCommandSourceIdentity(t *testing.T) {
	frame, err := buildGCSHeartbeatFrame(config{
		SourceSystem:    250,
		SourceComponent: 191,
	})
	if err != nil {
		t.Fatalf("build heartbeat: %v", err)
	}
	packets, err := mavcodec.NewParser().Parse(frame)
	if err != nil {
		t.Fatalf("parse heartbeat: %v", err)
	}
	if len(packets) != 1 {
		t.Fatalf("packets = %d, want 1", len(packets))
	}
	packet := packets[0]
	if packet.MessageID != mavcodec.MessageIDHeartbeat ||
		packet.SystemID != 250 ||
		packet.ComponentID != 191 {
		t.Fatalf("heartbeat packet = %+v", packet)
	}
	if packet.ParsedFields["type"] != mavcodec.TypeGCS ||
		packet.ParsedFields["autopilot"] != mavcodec.AutopilotInvalid {
		t.Fatalf("heartbeat fields = %+v", packet.ParsedFields)
	}
}

func TestRouteFromRawFrameLearnsTargetSystemRemoteAddr(t *testing.T) {
	decoder := newRouteLearningDecoder(t)
	parser := mavcodec.NewParser()
	frame := mustHeartbeatFrame(t, 1, 1)
	wire := mustRawFrameMessage(t, "172.18.0.9:18570", frame)

	route, ok, err := routeFromRawFrame(wire, decoder, parser, 1)
	if err != nil {
		t.Fatalf("learn route from raw frame: %v", err)
	}
	if !ok {
		t.Fatal("expected route to be learned")
	}
	if route != "172.18.0.9:18570" {
		t.Fatalf("route = %q", route)
	}
}

func TestRouteFromRawFrameIgnoresOtherSystem(t *testing.T) {
	decoder := newRouteLearningDecoder(t)
	parser := mavcodec.NewParser()
	frame := mustHeartbeatFrame(t, 42, 1)
	wire := mustRawFrameMessage(t, "172.18.0.42:18570", frame)

	route, ok, err := routeFromRawFrame(wire, decoder, parser, 1)
	if err != nil {
		t.Fatalf("learn route from other system: %v", err)
	}
	if ok || route != "" {
		t.Fatalf("route = %q ok=%v, want no route", route, ok)
	}
}

func TestRunForwardsSimulatorReplies(t *testing.T) {
	simulator, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen simulator UDP: %v", err)
	}
	defer simulator.Close()
	forwardSink, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen forward sink UDP: %v", err)
	}
	defer forwardSink.Close()

	reply := []byte("px4-command-ack-frame")
	simulatorDone := make(chan error, 1)
	go func() {
		buffer := make([]byte, 512)
		if err := simulator.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			simulatorDone <- err
			return
		}
		_, addr, err := simulator.ReadFrom(buffer)
		if err != nil {
			simulatorDone <- err
			return
		}
		_, err = simulator.WriteTo(reply, addr)
		simulatorDone <- err
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = run([]string{
		"-confirm-simulator-only",
		"-route", simulator.LocalAddr().String(),
		"-forward-replies-to", forwardSink.LocalAddr().String(),
		"-timeout", "200ms",
		"-reply-timeout", "200ms",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run command with reply forwarding: %v\nstderr: %s", err, stderr.String())
	}
	if err := <-simulatorDone; err != nil {
		t.Fatalf("simulator reply: %v", err)
	}

	buffer := make([]byte, 512)
	if err := forwardSink.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set forward sink deadline: %v", err)
	}
	n, _, err := forwardSink.ReadFrom(buffer)
	if err != nil {
		t.Fatalf("read forwarded reply: %v\nstdout: %s", err, stdout.String())
	}
	if got := string(buffer[:n]); got != string(reply) {
		t.Fatalf("forwarded reply = %q, want %q", got, string(reply))
	}
	output := stdout.String()
	if !strings.Contains(output, "forwarded_reply_bytes=") ||
		!strings.Contains(output, "forwarded_replies=1") {
		t.Fatalf("forwarding output missing counters:\n%s", output)
	}
}

func TestRunRetriesCommandUntilDirectCommandAck(t *testing.T) {
	simulator, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen simulator UDP: %v", err)
	}
	defer simulator.Close()

	seen := make(chan *mavcodec.Packet, 2)
	simulatorDone := make(chan error, 1)
	go func() {
		parser := mavcodec.NewParser()
		buffer := make([]byte, 512)
		for attempt := 1; attempt <= 2; attempt++ {
			if err := simulator.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				simulatorDone <- err
				return
			}
			n, addr, err := simulator.ReadFrom(buffer)
			if err != nil {
				simulatorDone <- err
				return
			}
			packets, err := parser.Parse(buffer[:n])
			if err != nil {
				simulatorDone <- err
				return
			}
			var command *mavcodec.Packet
			for _, packet := range packets {
				if packet.MessageID == mavcodec.MessageIDCommandLong {
					command = packet
					break
				}
			}
			if command == nil {
				simulatorDone <- errExpectedCommandLong
				return
			}
			seen <- command
			if attempt == 2 {
				ack, err := mavcodec.NewGenerator(1, 1).GenerateCommandAck(mavcodec.CommandAckMessage{
					Command:           mavcodec.CommandRequestMessage,
					Result:            mavcodec.MAVResultAccepted,
					TargetSystemID:    255,
					TargetComponentID: 190,
				})
				if err != nil {
					simulatorDone <- err
					return
				}
				_, err = simulator.WriteTo(ack, addr)
				simulatorDone <- err
				return
			}
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = run([]string{
		"-confirm-simulator-only",
		"-route", simulator.LocalAddr().String(),
		"-attempts", "3",
		"-retry-interval", "30ms",
		"-reply-timeout", "200ms",
		"-timeout", "200ms",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run command with direct ACK retry: %v\nstderr: %s", err, stderr.String())
	}
	if err := <-simulatorDone; err != nil {
		t.Fatalf("simulator command ACK: %v", err)
	}

	first := <-seen
	second := <-seen
	if first.Sequence != 1 || second.Sequence != 2 {
		t.Fatalf("command sequences = %d/%d, want 1/2", first.Sequence, second.Sequence)
	}
	if got, _ := packetField[uint8](first, "confirmation"); got != 0 {
		t.Fatalf("first confirmation = %d, want 0", got)
	}
	if got, _ := packetField[uint8](second, "confirmation"); got != 1 {
		t.Fatalf("second confirmation = %d, want 1", got)
	}

	output := stdout.String()
	for _, want := range []string{
		"retrying_without_direct_command_ack attempt=1 next_attempt=2",
		"observed_command_ack command=512 result=accepted source=1/1 target=255/190",
		"command_attempts=2",
		"direct_command_acks=1",
		"last_ack_result=accepted",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("retry output missing %q:\n%s", want, output)
		}
	}
}

func TestCommandForActionOnlyAllowsReadSideMVPCommand(t *testing.T) {
	command, params, err := commandForAction(actionRequestAutopilotVersion)
	if err != nil {
		t.Fatalf("command for action: %v", err)
	}
	if command != mavcodec.CommandRequestMessage {
		t.Fatalf("command = %d, want %d", command, mavcodec.CommandRequestMessage)
	}
	if params[0] != float32(mavcodec.MessageIDAutopilotVersion) {
		t.Fatalf("param1 = %f, want %d", params[0], mavcodec.MessageIDAutopilotVersion)
	}

	if _, _, err := commandForAction("arm"); err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func newRouteLearningDecoder(t *testing.T) *message.Decoder {
	t.Helper()
	registry := payloadregistry.New()
	if err := mavcomponent.RegisterPayloads(registry); err != nil {
		t.Fatalf("register MAVLink payloads: %v", err)
	}
	return message.NewDecoder(registry)
}

func mustHeartbeatFrame(t *testing.T, systemID, componentID uint8) []byte {
	t.Helper()
	frame, err := mavcodec.NewGenerator(systemID, componentID).GenerateHeartbeat(mavcodec.HeartbeatMessage{
		VehicleType:    mavcodec.TypeQuadrotor,
		Autopilot:      mavcodec.AutopilotPX4,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: 3,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	return frame
}

func mustRawFrameMessage(t *testing.T, remoteAddr string, frame []byte) []byte {
	t.Helper()
	now := time.Date(2026, 6, 27, 20, 0, 0, 0, time.UTC)
	payload := mavcomponent.NewRawFramePayload("udp:test", remoteAddr, now, frame)
	envelope := message.NewBaseMessage(
		mavcomponent.RawFrameType,
		payload,
		"semops-input-mavlink-udp",
		message.WithTime(now),
	)
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal raw frame BaseMessage: %v", err)
	}
	return data
}

func TestNormalizeUDPRouteRejectsListenStyleRoute(t *testing.T) {
	if _, err := normalizeUDPRoute("udp://:14540"); err == nil {
		t.Fatal("expected listen-style route error")
	}

	got, err := normalizeUDPRoute("udp://127.0.0.1:14540")
	if err != nil {
		t.Fatalf("normalize route: %v", err)
	}
	if got != "127.0.0.1:14540" {
		t.Fatalf("route = %q", got)
	}
}
