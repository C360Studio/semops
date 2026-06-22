package klv

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
)

func TestDemuxComponentPublishesPacketFromMediaRefMessage(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	packetBytes := buildMISB0601Packet(
		misbField(misbTagPrecisionTimeStamp, beU64(uint64(time.Date(2026, 6, 22, 17, 0, 0, 0, time.UTC).UnixMicro()))),
		misbField(misbTagPlatformDesignation, []byte("SYNTHETIC-UAS-1")),
	)
	runner := &recordingCommandRunner{
		ffprobeOutput: []byte(`{"streams":[{"index":0,"codec_type":"video"},{"index":2,"codec_type":"data","codec_tag_string":"KLVA"}]}`),
		ffmpegOutput:  packetBytes,
	}
	bus := &klvRecordingBus{}
	clockTime := time.Date(2026, 6, 22, 17, 1, 0, 0, time.UTC)
	demux, err := NewDemuxComponent(DemuxConfig{
		Registry: registry,
		Bus:      bus,
		Runner:   runner,
		Clock:    func() time.Time { return clockTime },
	})
	if err != nil {
		t.Fatalf("new demux: %v", err)
	}
	if err := demux.Initialize(); err != nil {
		t.Fatalf("initialize demux: %v", err)
	}

	media := NewMediaRefPayload("klv:fixture", "file:///fixtures/klv/demo.ts", "object://semops/klv/demo.ts", clockTime)
	media.ContentHash = "sha256:abcdef"
	mediaWire, err := marshalBaseMessage(MediaRefType, media, "semops-input-klv-media-ref", clockTime)
	if err != nil {
		t.Fatalf("marshal media-ref: %v", err)
	}

	if err := demux.HandleMediaRefMessage(context.Background(), mediaWire); err != nil {
		t.Fatalf("handle media-ref message: %v", err)
	}
	if len(bus.published) != 1 {
		t.Fatalf("published %d messages, want 1", len(bus.published))
	}
	published := bus.published[0]
	if published.subject != DefaultPacketSubject {
		t.Fatalf("published subject = %q, want %q", published.subject, DefaultPacketSubject)
	}
	if published.subject == SubjectEntityCreateWithTriples || published.subject == SubjectEntityUpdateWithTriples {
		t.Fatalf("demux published graph mutation subject %q", published.subject)
	}

	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published packet: %v", err)
	}
	packet, ok := envelope.Payload().(*PacketPayload)
	if !ok {
		t.Fatalf("published payload = %T, want *PacketPayload", envelope.Payload())
	}
	if packet.Source != DefaultDemuxSource {
		t.Fatalf("packet source = %q, want %q", packet.Source, DefaultDemuxSource)
	}
	if packet.MediaRef != media.StorageRef {
		t.Fatalf("packet media_ref = %q, want %q", packet.MediaRef, media.StorageRef)
	}
	if packet.StreamIndex != 2 {
		t.Fatalf("packet stream index = %d, want 2", packet.StreamIndex)
	}
	if !reflect.DeepEqual(packet.PacketBytes, packetBytes) {
		t.Fatalf("packet bytes = %x, want %x", packet.PacketBytes, packetBytes)
	}
	if !strings.HasPrefix(packet.PacketRef, "klv://packet/abcdef/2/") {
		t.Fatalf("packet ref = %q, want hash and stream index prefix", packet.PacketRef)
	}
	if got := demux.DataFlow().MessagesPerSecond; got <= 0 {
		t.Fatalf("demux messages per second = %f, want > 0", got)
	}

	requireCommand(t, runner.calls, DefaultFFprobePath, []string{
		"-v", "error",
		"-select_streams", "d",
		"-show_entries", "stream=index,codec_type,codec_tag_string,codec_name",
		"-of", "json",
		"/fixtures/klv/demo.ts",
	})
	requireCommand(t, runner.calls, DefaultFFmpegPath, []string{
		"-v", "error",
		"-i", "/fixtures/klv/demo.ts",
		"-map", "0:2",
		"-c", "copy",
		"-f", "data",
		"-",
	})
}

func TestDemuxComponentSubscribesWhenBusIsConfigured(t *testing.T) {
	bus := &klvRecordingBus{}
	demux, err := NewDemuxComponent(DemuxConfig{
		Bus: bus,
		Runner: &recordingCommandRunner{
			ffprobeOutput: []byte(`{"streams":[]}`),
		},
	})
	if err != nil {
		t.Fatalf("new demux: %v", err)
	}
	if err := demux.Start(context.Background()); err != nil {
		t.Fatalf("start demux: %v", err)
	}
	if _, ok := bus.handlers[DefaultMediaRefSubject]; !ok {
		t.Fatalf("demux did not subscribe to %q", DefaultMediaRefSubject)
	}
	if err := demux.Stop(time.Second); err != nil {
		t.Fatalf("stop demux: %v", err)
	}
}

func TestDemuxRejectsStorageOnlyAndRemoteMedia(t *testing.T) {
	runner := &recordingCommandRunner{}
	demux, err := NewDemuxComponent(DemuxConfig{
		Bus:    &klvRecordingBus{},
		Runner: runner,
	})
	if err != nil {
		t.Fatalf("new demux: %v", err)
	}
	now := time.Now().UTC()
	tests := []struct {
		name  string
		media *MediaRefPayload
		want  string
	}{
		{
			name:  "storage only",
			media: NewMediaRefPayload("klv:fixture", "", "object://semops/klv/demo.ts", now),
			want:  "local file URI is required",
		},
		{
			name:  "remote uri",
			media: NewMediaRefPayload("klv:fixture", "https://example.test/demo.ts", "", now),
			want:  "unsupported URI scheme",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := demux.HandleMediaRefPayload(context.Background(), tt.media); err == nil {
				t.Fatal("expected demux to fail")
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %#v, want none for rejected media", runner.calls)
	}
}

func TestDemuxRejectsMissingDataStreamAndOversizedOutput(t *testing.T) {
	now := time.Now().UTC()
	media := NewMediaRefPayload("klv:fixture", "file:///fixtures/klv/demo.ts", "", now)
	tests := []struct {
		name   string
		runner *recordingCommandRunner
		max    int
		want   string
	}{
		{
			name:   "no data stream",
			runner: &recordingCommandRunner{ffprobeOutput: []byte(`{"streams":[{"index":0,"codec_type":"video"}]}`)},
			max:    64,
			want:   "no data stream found",
		},
		{
			name: "oversized output",
			runner: &recordingCommandRunner{
				ffprobeOutput: []byte(`{"streams":[{"index":3,"codec_type":"data"}]}`),
				ffmpegOutput:  []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			},
			max:  4,
			want: "output exceeds max_packet_bytes=4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			demux, err := NewDemuxComponent(DemuxConfig{
				Bus:            &klvRecordingBus{},
				Runner:         tt.runner,
				MaxPacketBytes: tt.max,
			})
			if err != nil {
				t.Fatalf("new demux: %v", err)
			}
			if err := demux.HandleMediaRefPayload(context.Background(), media); err == nil {
				t.Fatal("expected demux to fail")
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

type recordingCommandRunner struct {
	calls         []commandCall
	ffprobeOutput []byte
	ffmpegOutput  []byte
	ffprobeErr    error
	ffmpegErr     error
}

type commandCall struct {
	name           string
	args           []string
	maxStdoutBytes int
}

func (r *recordingCommandRunner) Run(_ context.Context, name string, args []string, maxStdoutBytes int) ([]byte, error) {
	r.calls = append(r.calls, commandCall{
		name:           name,
		args:           append([]string(nil), args...),
		maxStdoutBytes: maxStdoutBytes,
	})
	switch name {
	case DefaultFFprobePath:
		return append([]byte(nil), r.ffprobeOutput...), r.ffprobeErr
	case DefaultFFmpegPath:
		if r.ffmpegErr != nil {
			return nil, r.ffmpegErr
		}
		if len(r.ffmpegOutput) > maxStdoutBytes {
			return nil, errCommandOutputTooLarge
		}
		return append([]byte(nil), r.ffmpegOutput...), nil
	default:
		return nil, errors.New("unexpected command")
	}
}

func requireCommand(t *testing.T, calls []commandCall, name string, args []string) {
	t.Helper()
	for _, call := range calls {
		if call.name == name && reflect.DeepEqual(call.args, args) {
			return
		}
	}
	t.Fatalf("missing command %s %#v in calls %#v", name, args, calls)
}
