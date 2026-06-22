package klv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

const (
	envKLVPublicSamplePath            = "SEMOPS_KLV_PUBLIC_SAMPLE_PATH"
	envKLVPublicSampleSourceURL       = "SEMOPS_KLV_PUBLIC_SAMPLE_SOURCE_URL"
	envKLVPublicSampleProvenance      = "SEMOPS_KLV_PUBLIC_SAMPLE_PROVENANCE"
	envKLVPublicSampleMaxExtractBytes = "SEMOPS_KLV_PUBLIC_SAMPLE_MAX_EXTRACT_BYTES"
	envKLVPublicSampleMaxPacketBytes  = "SEMOPS_KLV_PUBLIC_SAMPLE_MAX_PACKET_BYTES"
	envKLVPublicSampleMaxPackets      = "SEMOPS_KLV_PUBLIC_SAMPLE_MAX_PACKETS"
)

func TestPublicKLVSampleSmokeWithLocalPath(t *testing.T) {
	samplePath := os.Getenv(envKLVPublicSamplePath)
	if samplePath == "" {
		t.Skipf("%s is not set; skipping opt-in public KLV sample smoke", envKLVPublicSamplePath)
	}
	sourceURL := os.Getenv(envKLVPublicSampleSourceURL)
	provenance := os.Getenv(envKLVPublicSampleProvenance)
	if sourceURL == "" || provenance == "" {
		t.Fatalf(
			"%s and %s are required when %s is set",
			envKLVPublicSampleSourceURL,
			envKLVPublicSampleProvenance,
			envKLVPublicSamplePath,
		)
	}
	samplePath, err := resolvePublicSamplePath(samplePath)
	if err != nil {
		t.Fatalf("resolve public KLV sample path: %v", err)
	}
	ffmpegPath, ok := requireToolForTest(t, DefaultFFmpegPath)
	if !ok {
		return
	}
	ffprobePath, ok := requireToolForTest(t, DefaultFFprobePath)
	if !ok {
		return
	}

	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	demuxBus := &klvRecordingBus{}
	maxPacketBytes := intEnvForTest(t, envKLVPublicSampleMaxPacketBytes, DefaultMaxPacketBytes)
	demux, err := NewDemuxComponent(DemuxConfig{
		Registry:        registry,
		Bus:             demuxBus,
		Runner:          OSCommandRunner{},
		FFmpegPath:      ffmpegPath,
		FFprobePath:     ffprobePath,
		MaxExtractBytes: intEnvForTest(t, envKLVPublicSampleMaxExtractBytes, 8*1024*1024),
		MaxPacketBytes:  maxPacketBytes,
		MaxPackets:      intEnvForTest(t, envKLVPublicSampleMaxPackets, 4096),
		Clock:           time.Now,
	})
	if err != nil {
		t.Fatalf("new demux: %v", err)
	}
	if err := demux.Initialize(); err != nil {
		t.Fatalf("initialize demux: %v", err)
	}
	receivedAt := time.Now().UTC()
	media := NewMediaRefPayload(
		"klv:public-sample",
		fileURIForTest(samplePath),
		"public-sample://klv/"+sanitizePublicSampleName(filepath.Base(samplePath)),
		receivedAt,
	)
	media.MediaType = "video/mpeg"
	media.FixtureKind = "public-sample-smoke"
	media.Provenance = strings.TrimSpace(sourceURL + " | " + provenance)

	if err := demux.HandleMediaRefPayload(context.Background(), media); err != nil {
		t.Fatalf("demux public KLV sample: %v", err)
	}
	if len(demuxBus.published) == 0 {
		t.Fatal("public KLV sample demux published no packet messages")
	}

	decoderBus := &klvRecordingBus{}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Registry:       registry,
		Bus:            decoderBus,
		MaxPacketBytes: maxPacketBytes,
	})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}
	var decodeErrors []string
	for _, published := range demuxBus.published {
		if published.subject != DefaultPacketSubject {
			continue
		}
		if err := decoder.HandlePacketMessage(context.Background(), published.data); err != nil {
			decodeErrors = append(decodeErrors, err.Error())
		}
	}
	if len(decoderBus.published) == 0 {
		t.Fatalf("public KLV sample produced no decoded frames; packet_count=%d decode_errors=%v", len(demuxBus.published), firstStrings(decodeErrors, 5))
	}

	frames := decodeFramePayloads(t, registry, decoderBus.published)
	if !hasPublicSampleGeometry(frames) {
		t.Fatalf("decoded %d public sample frames but none contained supported sensor or frame-center geometry", len(frames))
	}
}

func intEnvForTest(t *testing.T, name string, defaultValue int) int {
	t.Helper()
	raw := os.Getenv(name)
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		t.Fatalf("%s=%q must be a positive integer", name, raw)
	}
	return value
}

func resolvePublicSamplePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("stat %q: %w", path, os.ErrNotExist)
	}
	repoRootPath := filepath.Join("..", "..", "..", path)
	if _, err := os.Stat(repoRootPath); err != nil {
		return "", fmt.Errorf("stat %q or repo-root relative %q: %w", path, repoRootPath, err)
	}
	absPath, err := filepath.Abs(repoRootPath)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %q: %w", repoRootPath, err)
	}
	return absPath, nil
}

func decodeFramePayloads(t *testing.T, registry *payloadregistry.Registry, published []klvPublishedMessage) []*MISB0601FramePayload {
	t.Helper()
	decoder := message.NewDecoder(registry)
	frames := make([]*MISB0601FramePayload, 0, len(published))
	for index, msg := range published {
		if msg.subject != DefaultFrameSubject {
			continue
		}
		envelope, err := decoder.Decode(msg.data)
		if err != nil {
			t.Fatalf("decode public sample frame envelope %d: %v", index, err)
		}
		frame, ok := envelope.Payload().(*MISB0601FramePayload)
		if !ok {
			t.Fatalf("public sample frame payload %d = %T, want *MISB0601FramePayload", index, envelope.Payload())
		}
		frames = append(frames, frame)
	}
	return frames
}

func hasPublicSampleGeometry(frames []*MISB0601FramePayload) bool {
	for _, frame := range frames {
		if frame.SensorLatitude != nil || frame.SensorLongitude != nil ||
			frame.FrameCenterLatitude != nil || frame.FrameCenterLongitude != nil {
			return true
		}
	}
	return false
}

func sanitizePublicSampleName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}

func firstStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return append([]string(nil), values...)
	}
	return append([]string(nil), values[:limit]...)
}
