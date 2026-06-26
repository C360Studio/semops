package scenario

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	cotadapter "github.com/c360studio/semops/internal/adapters/cot"
	mavadapter "github.com/c360studio/semops/internal/adapters/mavlink"
)

const defaultFeedWriteTimeout = time.Second

type UDPFeedSinkConfig struct {
	Addr         string
	Source       string
	WriteTimeout time.Duration
	Clock        func() time.Time
}

type UDPFeedHealth struct {
	Source      string
	Ready       bool
	Messages    uint64
	Bytes       uint64
	LastWriteAt time.Time
	LastError   string
}

type MAVLinkUDPSink struct {
	cfg UDPFeedSinkConfig

	mu     sync.RWMutex
	health UDPFeedHealth
}

type CoTUDPSink struct {
	cfg UDPFeedSinkConfig

	mu     sync.RWMutex
	health UDPFeedHealth
}

func NewMAVLinkUDPSink(cfg UDPFeedSinkConfig) (*MAVLinkUDPSink, error) {
	cfg, err := normalizeUDPFeedSinkConfig(cfg, "mavlink:scenario:udp")
	if err != nil {
		return nil, err
	}
	return &MAVLinkUDPSink{
		cfg:    cfg,
		health: UDPFeedHealth{Source: cfg.Source, Ready: true},
	}, nil
}

func NewCoTUDPSink(cfg UDPFeedSinkConfig) (*CoTUDPSink, error) {
	cfg, err := normalizeUDPFeedSinkConfig(cfg, "tak-cot:scenario:udp")
	if err != nil {
		return nil, err
	}
	return &CoTUDPSink{
		cfg:    cfg,
		health: UDPFeedHealth{Source: cfg.Source, Ready: true},
	}, nil
}

func (s *MAVLinkUDPSink) IngestFrame(ctx context.Context, frame []byte) (mavadapter.IngestResult, error) {
	if s == nil {
		return mavadapter.IngestResult{}, fmt.Errorf("mavlink UDP scenario sink is nil")
	}
	if err := writeUDP(ctx, s.cfg, frame); err != nil {
		s.recordError(err)
		return mavadapter.IngestResult{}, err
	}
	s.recordWrite(len(frame))
	return mavadapter.IngestResult{RawRef: "udp://" + s.cfg.Addr}, nil
}

func (s *MAVLinkUDPSink) Health() mavadapter.Health {
	if s == nil {
		return mavadapter.Health{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return mavadapter.Health{
		Source:         s.health.Source,
		Ready:          s.health.Ready,
		FramesReceived: s.health.Messages,
		FramesCaptured: s.health.Messages,
		LastFrameAt:    s.health.LastWriteAt,
		LastRawRef:     "udp://" + s.cfg.Addr,
		LastError:      s.health.LastError,
	}
}

func (s *CoTUDPSink) IngestEvent(ctx context.Context, rawXML []byte) (cotadapter.IngestResult, error) {
	if s == nil {
		return cotadapter.IngestResult{}, fmt.Errorf("CoT UDP scenario sink is nil")
	}
	if err := writeUDP(ctx, s.cfg, rawXML); err != nil {
		s.recordError(err)
		return cotadapter.IngestResult{}, err
	}
	s.recordWrite(len(rawXML))
	return cotadapter.IngestResult{RawRef: "udp://" + s.cfg.Addr}, nil
}

func (s *CoTUDPSink) Health() cotadapter.Health {
	if s == nil {
		return cotadapter.Health{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cotadapter.Health{
		Source:         s.health.Source,
		Ready:          s.health.Ready,
		EventsReceived: s.health.Messages,
		EventsCaptured: s.health.Messages,
		LastEventAt:    s.health.LastWriteAt,
		LastRawRef:     "udp://" + s.cfg.Addr,
		LastError:      s.health.LastError,
	}
}

func (s *MAVLinkUDPSink) recordWrite(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.Messages++
	s.health.Bytes += uint64(size)
	s.health.LastWriteAt = s.cfg.Clock().UTC()
	s.health.LastError = ""
}

func (s *MAVLinkUDPSink) recordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.LastError = err.Error()
}

func (s *CoTUDPSink) recordWrite(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.Messages++
	s.health.Bytes += uint64(size)
	s.health.LastWriteAt = s.cfg.Clock().UTC()
	s.health.LastError = ""
}

func (s *CoTUDPSink) recordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.LastError = err.Error()
}

func normalizeUDPFeedSinkConfig(cfg UDPFeedSinkConfig, defaultSource string) (UDPFeedSinkConfig, error) {
	if cfg.Addr == "" {
		return UDPFeedSinkConfig{}, fmt.Errorf("scenario UDP sink requires an address")
	}
	if cfg.Source == "" {
		cfg.Source = defaultSource
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = defaultFeedWriteTimeout
	}
	if cfg.WriteTimeout < 0 {
		return UDPFeedSinkConfig{}, fmt.Errorf("scenario UDP sink write timeout must be greater than zero")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return cfg, nil
}

func writeUDP(ctx context.Context, cfg UDPFeedSinkConfig, payload []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(payload) == 0 {
		return fmt.Errorf("scenario UDP sink refuses empty payload")
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("dial scenario UDP sink %s: %w", cfg.Addr, err)
	}
	defer conn.Close()
	if cfg.WriteTimeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout)); err != nil {
			return fmt.Errorf("set scenario UDP sink write deadline: %w", err)
		}
	}
	written, err := conn.Write(payload)
	if err != nil {
		return fmt.Errorf("write scenario UDP payload to %s: %w", cfg.Addr, err)
	}
	if written != len(payload) {
		return fmt.Errorf("write scenario UDP payload to %s: wrote %d of %d bytes", cfg.Addr, written, len(payload))
	}
	return nil
}

var _ MAVLinkSink = (*MAVLinkUDPSink)(nil)
var _ CoTSink = (*CoTUDPSink)(nil)
