package cot

import (
	"context"
	"fmt"
	"sync"
	"time"

	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
)

type ReplayAppender interface {
	Append(record cotcodec.RawEventRecord) error
}

type Config struct {
	Source  string
	RawLane *cotcodec.RawLane
	Replay  ReplayAppender
	Clock   func() time.Time
	OnEvent func(IngestResult)
}

type Adapter struct {
	rawLane *cotcodec.RawLane
	replay  ReplayAppender
	clock   func() time.Time
	onEvent func(IngestResult)

	mu     sync.RWMutex
	health Health
}

type IngestResult struct {
	RawRef string
	Event  cotcodec.Event
}

type Health struct {
	Source         string
	Ready          bool
	EventsReceived uint64
	EventsCaptured uint64
	EventsDecoded  uint64
	ParseErrors    uint64
	CaptureErrors  uint64
	ReplayErrors   uint64
	LastEventAt    time.Time
	LastRawRef     string
	LastUID        string
	LastType       string
	LastCallsign   string
	LastError      string
}

func NewAdapter(cfg Config) (*Adapter, error) {
	if cfg.Source == "" {
		cfg.Source = "tak-cot"
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.RawLane == nil {
		cfg.RawLane = cotcodec.NewRawLane(cotcodec.RawLaneConfig{Source: cfg.Source, Clock: cfg.Clock})
	}
	return &Adapter{
		rawLane: cfg.RawLane,
		replay:  cfg.Replay,
		clock:   cfg.Clock,
		onEvent: cfg.OnEvent,
		health: Health{
			Source: cfg.Source,
			Ready:  true,
		},
	}, nil
}

func (a *Adapter) IngestEvent(ctx context.Context, raw []byte) (IngestResult, error) {
	if a == nil {
		return IngestResult{}, fmt.Errorf("cot adapter is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return IngestResult{}, err
	}

	now := a.clock().UTC()
	a.recordReceived(now)

	event, parseErr := cotcodec.Unmarshal(raw)
	if parseErr != nil {
		record, captureErr := a.rawLane.Capture(raw, nil)
		if captureErr != nil {
			a.recordCaptureError(captureErr)
			return IngestResult{}, fmt.Errorf("capture unparsable cot event: %w", captureErr)
		}
		if err := a.appendReplay(record); err != nil {
			a.recordReplayError(err)
			return IngestResult{RawRef: record.Ref}, err
		}
		err := fmt.Errorf("parse cot event: %w", parseErr)
		a.recordParseError(err, record.Ref)
		return IngestResult{RawRef: record.Ref}, err
	}

	record, err := a.rawLane.Capture(raw, &event)
	if err != nil {
		a.recordCaptureError(err)
		return IngestResult{}, fmt.Errorf("capture cot event: %w", err)
	}
	if err := a.appendReplay(record); err != nil {
		a.recordReplayError(err)
		return IngestResult{RawRef: record.Ref, Event: event}, err
	}

	result := IngestResult{RawRef: record.Ref, Event: event}
	a.recordDecoded(record.Ref, event)
	if a.onEvent != nil {
		a.onEvent(result)
	}
	return result, nil
}

func (a *Adapter) Health() Health {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.health
}

func (a *Adapter) RawLane() *cotcodec.RawLane {
	return a.rawLane
}

func (a *Adapter) appendReplay(record cotcodec.RawEventRecord) error {
	if a.replay == nil {
		return nil
	}
	if err := a.replay.Append(record); err != nil {
		return fmt.Errorf("append cot replay record %q: %w", record.Ref, err)
	}
	return nil
}

func (a *Adapter) recordReceived(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.EventsReceived++
	a.health.LastEventAt = now
}

func (a *Adapter) recordDecoded(rawRef string, event cotcodec.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = true
	a.health.EventsCaptured++
	a.health.EventsDecoded++
	a.health.LastRawRef = rawRef
	a.health.LastUID = event.UID
	a.health.LastType = event.Type
	a.health.LastCallsign = event.Callsign
	a.health.LastError = ""
}

func (a *Adapter) recordParseError(err error, rawRef string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = false
	a.health.EventsCaptured++
	a.health.ParseErrors++
	a.health.LastRawRef = rawRef
	a.health.LastError = err.Error()
}

func (a *Adapter) recordCaptureError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = false
	a.health.CaptureErrors++
	a.health.LastError = err.Error()
}

func (a *Adapter) recordReplayError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = false
	a.health.ReplayErrors++
	a.health.LastError = err.Error()
}
