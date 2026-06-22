package klv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultDemuxSource         = "klv:demux"
	DefaultFFmpegPath          = "ffmpeg"
	DefaultFFprobePath         = "ffprobe"
	DefaultProbeOutputMaxBytes = 64 * 1024
	DefaultExtractOutputSlop   = 1
	ffprobeDataStreamSelection = "d"
	ffprobeStreamEntries       = "stream=index,codec_type,codec_tag_string,codec_name"
	ffprobeOutputFormat        = "json"
	ffmpegExtractOutputFormat  = "data"
	ffmpegCopyCodec            = "copy"
	ffmpegQuietLogLevel        = "error"
	ffprobeQuietLogLevel       = "error"
)

var errCommandOutputTooLarge = errors.New("command output exceeded configured byte limit")

type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, maxStdoutBytes int) ([]byte, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) Run(ctx context.Context, name string, args []string, maxStdoutBytes int) ([]byte, error) {
	if maxStdoutBytes <= 0 {
		return nil, fmt.Errorf("run %s: max stdout bytes must be greater than zero", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout boundedBuffer
	stdout.limit = maxStdoutBytes
	var stderr boundedBuffer
	stderr.limit = 8 * 1024
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stdout.err != nil {
			return nil, stdout.err
		}
		return nil, fmt.Errorf("run %s %s: %w; stderr=%s", name, strings.Join(args, " "), err, stderr.String())
	}
	if stdout.err != nil {
		return nil, stdout.err
	}
	return stdout.Bytes(), nil
}

type boundedBuffer struct {
	buffer bytes.Buffer
	limit  int
	err    error
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		b.err = errCommandOutputTooLarge
		return 0, b.err
	}
	if b.buffer.Len()+len(p) > b.limit {
		remaining := b.limit - b.buffer.Len()
		if remaining > 0 {
			_, _ = b.buffer.Write(p[:remaining])
		}
		b.err = errCommandOutputTooLarge
		return 0, b.err
	}
	return b.buffer.Write(p)
}

func (b *boundedBuffer) Bytes() []byte {
	return append([]byte(nil), b.buffer.Bytes()...)
}

func (b *boundedBuffer) String() string {
	return b.buffer.String()
}

type ffprobeStreamsResponse struct {
	Streams []ffprobeStream `json:"streams"`
}

type ffprobeStream struct {
	Index          int    `json:"index"`
	CodecType      string `json:"codec_type"`
	CodecName      string `json:"codec_name"`
	CodecTagString string `json:"codec_tag_string"`
}

func demuxKLVPacket(ctx context.Context, cfg DemuxConfig, media *MediaRefPayload) (*PacketPayload, error) {
	if media == nil {
		return nil, errors.New("demux KLV media: media-ref payload is nil")
	}
	if err := media.Validate(); err != nil {
		return nil, fmt.Errorf("demux KLV media: %w", err)
	}
	mediaPath, err := localMediaPath(media)
	if err != nil {
		return nil, err
	}
	streamIndex, err := selectKLVDataStream(ctx, cfg, mediaPath)
	if err != nil {
		return nil, err
	}
	packetBytes, err := extractDataStream(ctx, cfg, mediaPath, streamIndex)
	if err != nil {
		return nil, err
	}
	now := cfg.Clock().UTC()
	packet := NewPacketPayload(cfg.Source, mediaReference(media), now, packetBytes)
	packet.PacketRef = packetRef(media, streamIndex, now)
	packet.StorageRef = ""
	packet.ReceivedAt = now
	packet.StreamIndex = streamIndex
	packet.PacketTime = media.ReceivedAt.UTC()
	if media.ByteRange != nil {
		packet.ByteOffset = media.ByteRange.Start
	}
	if err := packet.Validate(); err != nil {
		return nil, fmt.Errorf("demux KLV media: %w", err)
	}
	return packet, nil
}

func selectKLVDataStream(ctx context.Context, cfg DemuxConfig, mediaPath string) (int, error) {
	args := []string{
		"-v", ffprobeQuietLogLevel,
		"-select_streams", ffprobeDataStreamSelection,
		"-show_entries", ffprobeStreamEntries,
		"-of", ffprobeOutputFormat,
		mediaPath,
	}
	data, err := cfg.Runner.Run(ctx, cfg.FFprobePath, args, cfg.ProbeOutputMaxBytes)
	if err != nil {
		return 0, fmt.Errorf("probe KLV data streams: %w", err)
	}
	var response ffprobeStreamsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return 0, fmt.Errorf("parse ffprobe stream response: %w", err)
	}
	for _, stream := range response.Streams {
		if stream.CodecType == "data" {
			return stream.Index, nil
		}
	}
	return 0, errors.New("probe KLV data streams: no data stream found")
}

func extractDataStream(ctx context.Context, cfg DemuxConfig, mediaPath string, streamIndex int) ([]byte, error) {
	args := []string{
		"-v", ffmpegQuietLogLevel,
		"-i", mediaPath,
		"-map", fmt.Sprintf("0:%d", streamIndex),
		"-c", ffmpegCopyCodec,
		"-f", ffmpegExtractOutputFormat,
		"-",
	}
	limit := cfg.MaxPacketBytes + DefaultExtractOutputSlop
	data, err := cfg.Runner.Run(ctx, cfg.FFmpegPath, args, limit)
	if err != nil {
		if errors.Is(err, errCommandOutputTooLarge) {
			return nil, fmt.Errorf("extract KLV data stream: output exceeds max_packet_bytes=%d", cfg.MaxPacketBytes)
		}
		return nil, fmt.Errorf("extract KLV data stream: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("extract KLV data stream: no bytes extracted")
	}
	if len(data) > cfg.MaxPacketBytes {
		return nil, fmt.Errorf("extract KLV data stream: output exceeds max_packet_bytes=%d", cfg.MaxPacketBytes)
	}
	return data, nil
}

func localMediaPath(media *MediaRefPayload) (string, error) {
	if media.URI == "" {
		return "", errors.New("demux KLV media: local file URI is required; storage_ref-only demux is not implemented")
	}
	parsed, err := url.Parse(media.URI)
	if err != nil {
		return "", fmt.Errorf("demux KLV media URI %q: %w", media.URI, err)
	}
	if parsed.Scheme == "" {
		return filepath.Clean(media.URI), nil
	}
	if parsed.Scheme != "file" {
		return "", fmt.Errorf("demux KLV media: unsupported URI scheme %q", parsed.Scheme)
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		return "", fmt.Errorf("demux KLV media: unsupported file URI host %q", parsed.Host)
	}
	if parsed.Path == "" {
		return "", errors.New("demux KLV media: file URI path is required")
	}
	return filepath.Clean(parsed.Path), nil
}

func mediaReference(media *MediaRefPayload) string {
	if media.StorageRef != "" {
		return media.StorageRef
	}
	return media.URI
}

func packetRef(media *MediaRefPayload, streamIndex int, observedAt time.Time) string {
	base := media.ContentHash
	if base == "" {
		base = mediaReference(media)
	}
	base = strings.TrimPrefix(base, "sha256:")
	base = strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(base)
	return fmt.Sprintf("klv://packet/%s/%d/%s", base, streamIndex, observedAt.UTC().Format("20060102T150405.000000000Z"))
}
