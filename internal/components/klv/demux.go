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
	DefaultDemuxSource          = "klv:demux"
	DefaultFFmpegPath           = "ffmpeg"
	DefaultFFprobePath          = "ffprobe"
	DefaultProbeOutputMaxBytes  = 64 * 1024
	DefaultMaxPackets           = 64
	DefaultMaxMaterializedBytes = 32 * 1024 * 1024
	DefaultExtractOutputSlop    = 1
	ffprobeDataStreamSelection  = "d"
	ffprobeStreamEntries        = "stream=index,codec_type,codec_tag_string,codec_name"
	ffprobeOutputFormat         = "json"
	ffmpegExtractOutputFormat   = "data"
	ffmpegCopyCodec             = "copy"
	ffmpegQuietLogLevel         = "error"
	ffprobeQuietLogLevel        = "error"
)

var errCommandOutputTooLarge = errors.New("command output exceeded configured byte limit")

type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, maxStdoutBytes int) ([]byte, error)
}

type MediaMaterializer interface {
	MaterializeMedia(ctx context.Context, media *MediaRefPayload, maxBytes int) (MaterializedMedia, error)
}

type MaterializedMedia struct {
	Path      string
	SizeBytes int64
	Cleanup   func() error
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
	packets, err := demuxKLVPackets(ctx, cfg, media)
	if err != nil {
		return nil, err
	}
	if len(packets) != 1 {
		return nil, fmt.Errorf("demux KLV media: got %d packets, want 1", len(packets))
	}
	return packets[0], nil
}

func demuxKLVPackets(ctx context.Context, cfg DemuxConfig, media *MediaRefPayload) (packets []*PacketPayload, err error) {
	if media == nil {
		return nil, errors.New("demux KLV media: media-ref payload is nil")
	}
	if err := media.Validate(); err != nil {
		return nil, fmt.Errorf("demux KLV media: %w", err)
	}
	materialized, err := materializedMediaPath(ctx, cfg, media)
	if err != nil {
		return nil, err
	}
	if materialized.Cleanup != nil {
		defer func() {
			if cleanupErr := materialized.Cleanup(); err == nil && cleanupErr != nil {
				err = fmt.Errorf("cleanup materialized KLV media: %w", cleanupErr)
			}
		}()
	}
	streamIndex, err := selectKLVDataStream(ctx, cfg, materialized.Path)
	if err != nil {
		return nil, err
	}
	data, err := extractDataStream(ctx, cfg, materialized.Path, streamIndex)
	if err != nil {
		return nil, err
	}
	segments, err := splitMISB0601Packets(data, cfg.MaxPacketBytes, cfg.MaxPackets)
	if err != nil {
		return nil, fmt.Errorf("split KLV data stream: %w", err)
	}
	now := cfg.Clock().UTC()
	packets = make([]*PacketPayload, 0, len(segments))
	if media.ByteRange != nil {
		for index := range segments {
			segments[index].offset += int(media.ByteRange.Start)
		}
	}
	for _, segment := range segments {
		packet := NewPacketPayload(cfg.Source, mediaReference(media), now, segment.bytes)
		packet.PacketRef = packetRef(media, streamIndex, segment.index, now)
		packet.StorageRef = ""
		packet.ReceivedAt = now
		packet.StreamIndex = streamIndex
		packet.PacketTime = media.ReceivedAt.UTC()
		packet.ByteOffset = int64(segment.offset)
		if err := packet.Validate(); err != nil {
			return nil, fmt.Errorf("demux KLV media: %w", err)
		}
		packets = append(packets, packet)
	}
	return packets, nil
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
	limit := cfg.MaxExtractBytes + DefaultExtractOutputSlop
	data, err := cfg.Runner.Run(ctx, cfg.FFmpegPath, args, limit)
	if err != nil {
		if errors.Is(err, errCommandOutputTooLarge) {
			return nil, fmt.Errorf("extract KLV data stream: output exceeds max_extract_bytes=%d", cfg.MaxExtractBytes)
		}
		return nil, fmt.Errorf("extract KLV data stream: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("extract KLV data stream: no bytes extracted")
	}
	if len(data) > cfg.MaxExtractBytes {
		return nil, fmt.Errorf("extract KLV data stream: output exceeds max_extract_bytes=%d", cfg.MaxExtractBytes)
	}
	return data, nil
}

type klvPacketSegment struct {
	index  int
	offset int
	bytes  []byte
}

func splitMISB0601Packets(data []byte, maxPacketBytes int, maxPackets int) ([]klvPacketSegment, error) {
	if len(data) == 0 {
		return nil, errors.New("empty KLV data stream")
	}
	if maxPacketBytes <= 0 {
		return nil, errors.New("max_packet_bytes must be greater than zero")
	}
	if maxPackets <= 0 {
		return nil, errors.New("max_packets must be greater than zero")
	}
	segments := make([]klvPacketSegment, 0, 1)
	for offset := 0; offset < len(data); {
		if !bytes.HasPrefix(data[offset:], misb0601UASLocalSetKey) {
			return nil, fmt.Errorf("missing MISB ST 0601 universal key at byte %d", offset)
		}
		length, valueOffset, err := readBERLength(data, offset+len(misb0601UASLocalSetKey))
		if err != nil {
			return nil, fmt.Errorf("packet at byte %d length: %w", offset, err)
		}
		end := valueOffset + length
		if length < 0 || end > len(data) {
			return nil, fmt.Errorf("packet at byte %d local set length exceeds data stream", offset)
		}
		if end-offset > maxPacketBytes {
			return nil, fmt.Errorf("packet at byte %d exceeds max_packet_bytes=%d", offset, maxPacketBytes)
		}
		if len(segments) >= maxPackets {
			return nil, fmt.Errorf("packet count exceeds max_packets=%d", maxPackets)
		}
		segments = append(segments, klvPacketSegment{
			index:  len(segments),
			offset: offset,
			bytes:  append([]byte(nil), data[offset:end]...),
		})
		offset = end
	}
	return segments, nil
}

func materializedMediaPath(ctx context.Context, cfg DemuxConfig, media *MediaRefPayload) (MaterializedMedia, error) {
	if media.URI != "" {
		path, err := localMediaPath(media)
		if err != nil {
			return MaterializedMedia{}, err
		}
		return MaterializedMedia{Path: path}, nil
	}
	if media.StorageRef == "" {
		return MaterializedMedia{}, errors.New("demux KLV media: uri or storage_ref is required")
	}
	if cfg.Materializer == nil {
		return MaterializedMedia{}, errors.New("demux KLV media: storage_ref-only demux requires a bounded materializer")
	}
	materialized, err := cfg.Materializer.MaterializeMedia(ctx, media, cfg.MaxMaterializedBytes)
	if err != nil {
		return MaterializedMedia{}, fmt.Errorf("materialize KLV media storage_ref: %w", err)
	}
	if materialized.Path == "" {
		return MaterializedMedia{}, errors.New("materialize KLV media storage_ref: local path is required")
	}
	if cfg.MaxMaterializedBytes > 0 && materialized.SizeBytes > int64(cfg.MaxMaterializedBytes) {
		return MaterializedMedia{}, fmt.Errorf("materialize KLV media storage_ref: size %d exceeds max_materialized_bytes=%d", materialized.SizeBytes, cfg.MaxMaterializedBytes)
	}
	return materialized, nil
}

func localMediaPath(media *MediaRefPayload) (string, error) {
	if media.URI == "" {
		return "", errors.New("demux KLV media: local file URI is required for URI-backed demux")
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

func packetRef(media *MediaRefPayload, streamIndex int, packetIndex int, observedAt time.Time) string {
	base := media.ContentHash
	if base == "" {
		base = mediaReference(media)
	}
	base = strings.TrimPrefix(base, "sha256:")
	base = strings.NewReplacer("/", "_", ":", "_", " ", "_").Replace(base)
	return fmt.Sprintf(
		"klv://packet/%s/%d/%d/%s",
		base,
		streamIndex,
		packetIndex,
		observedAt.UTC().Format("20060102T150405.000000000Z"),
	)
}
