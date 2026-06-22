package klv

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

func TestDemuxDeterministicMPEGTSFixtureWithFFmpeg(t *testing.T) {
	ffmpegPath, ok := requireToolForTest(t, DefaultFFmpegPath)
	if !ok {
		return
	}
	ffprobePath, ok := requireToolForTest(t, DefaultFFprobePath)
	if !ok {
		return
	}

	truth := loadMISB0601TruthFixture(t)
	packet, err := truth.PacketPayload()
	if err != nil {
		t.Fatalf("build packet from truth fixture: %v", err)
	}
	tempDir := t.TempDir()
	packetPath := filepath.Join(tempDir, "deterministic.klv")
	if err := os.WriteFile(packetPath, packet.PacketBytes, 0o600); err != nil {
		t.Fatalf("write deterministic KLV packet fixture: %v", err)
	}
	mediaPath := filepath.Join(tempDir, "deterministic.ts")
	muxDeterministicKLVFixture(t, ffmpegPath, packetPath, mediaPath)

	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	bus := &klvRecordingBus{}
	clockTime := truth.ReceivedAt.Add(2 * time.Second)
	demux, err := NewDemuxComponent(DemuxConfig{
		Registry:       registry,
		Bus:            bus,
		Runner:         OSCommandRunner{},
		FFmpegPath:     ffmpegPath,
		FFprobePath:    ffprobePath,
		MaxPacketBytes: len(packet.PacketBytes) + DefaultExtractOutputSlop,
		Clock:          func() time.Time { return clockTime },
	})
	if err != nil {
		t.Fatalf("new demux: %v", err)
	}
	if err := demux.Initialize(); err != nil {
		t.Fatalf("initialize demux: %v", err)
	}
	media := NewMediaRefPayload("klv:fixture", fileURIForTest(mediaPath), truth.MediaRef, truth.ReceivedAt)
	media.FixtureKind = "deterministic-mpegts"
	media.Provenance = "generated locally from fixtures/klv/misb0601-truth.json"

	if err := demux.HandleMediaRefPayload(context.Background(), media); err != nil {
		t.Fatalf("demux deterministic MPEG-TS fixture: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("published %d messages, want 1", len(bus.published))
	}
	if bus.published[0].subject != DefaultPacketSubject {
		t.Fatalf("published subject = %q, want %q", bus.published[0].subject, DefaultPacketSubject)
	}
	envelope, err := message.NewDecoder(registry).Decode(bus.published[0].data)
	if err != nil {
		t.Fatalf("decode demuxed packet envelope: %v", err)
	}
	demuxed, ok := envelope.Payload().(*PacketPayload)
	if !ok {
		t.Fatalf("demuxed payload = %T, want *PacketPayload", envelope.Payload())
	}
	if demuxed.StreamIndex != 1 {
		t.Fatalf("demuxed stream index = %d, want 1", demuxed.StreamIndex)
	}
	if !bytes.Equal(demuxed.PacketBytes, packet.PacketBytes) {
		t.Fatalf("demuxed packet bytes differ from deterministic truth packet")
	}

	frame, err := DecodeMISB0601Packet(demuxed)
	if err != nil {
		t.Fatalf("decode demuxed deterministic MPEG-TS packet: %v", err)
	}
	if !frame.FrameTime.Equal(truth.FrameTime) {
		t.Fatalf("frame time = %s, want %s", frame.FrameTime, truth.FrameTime)
	}
	requireTruthClose(t, "sensor latitude", frame.SensorLatitude, truth.SensorLatitude, signed32QuantizationTolerance(90))
	requireTruthClose(t, "sensor longitude", frame.SensorLongitude, truth.SensorLongitude, signed32QuantizationTolerance(180))
	requireTruthClose(t, "frame center latitude", frame.FrameCenterLatitude, truth.FrameCenterLatitude, signed32QuantizationTolerance(90))
	requireTruthClose(t, "frame center longitude", frame.FrameCenterLongitude, truth.FrameCenterLongitude, signed32QuantizationTolerance(180))
}

func requireToolForTest(t *testing.T, name string) (string, bool) {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not found; skipping optional deterministic MPEG-TS fixture smoke", name)
		return "", false
	}
	return path, true
}

func muxDeterministicKLVFixture(t *testing.T, ffmpegPath string, packetPath string, mediaPath string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		ffmpegPath,
		"-v", "error",
		"-y",
		"-f", "lavfi",
		"-i", "testsrc=size=16x16:rate=1",
		"-f", "data",
		"-i", packetPath,
		"-map", "0:v:0",
		"-map", "1:0",
		"-c:v", "mpeg2video",
		"-c:d", "copy",
		"-t", "1",
		"-shortest",
		"-f", "mpegts",
		mediaPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mux deterministic MPEG-TS fixture: %v; output=%s", err, output)
	}
}

func fileURIForTest(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}
