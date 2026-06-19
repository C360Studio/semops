package cot

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semstreams/graph"
)

type ReplayAppender interface {
	Append(record cotcodec.RawEventRecord) error
}

type PlanWriter interface {
	Apply(ctx context.Context, plan cotprojector.Plan) error
}

type Config struct {
	Source    string
	RawLane   *cotcodec.RawLane
	Replay    ReplayAppender
	Projector *cotprojector.Projector
	Writer    PlanWriter
	Clock     func() time.Time
	OnEvent   func(IngestResult)
}

type Adapter struct {
	rawLane   *cotcodec.RawLane
	replay    ReplayAppender
	projector *cotprojector.Projector
	writer    PlanWriter
	clock     func() time.Time
	onEvent   func(IngestResult)

	mu     sync.RWMutex
	health Health
}

type IngestResult struct {
	RawRef    string
	Event     cotcodec.Event
	Mutations int
}

type Health struct {
	Source           string
	Ready            bool
	EventsReceived   uint64
	EventsCaptured   uint64
	EventsDecoded    uint64
	GraphMutations   uint64
	ParseErrors      uint64
	CaptureErrors    uint64
	ReplayErrors     uint64
	ProjectionDrops  uint64
	WriteErrors      uint64
	LastEventAt      time.Time
	LastGraphWriteAt time.Time
	LastRawRef       string
	LastUID          string
	LastType         string
	LastCallsign     string
	LastError        string
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
	if cfg.Projector != nil && cfg.Writer == nil {
		return nil, fmt.Errorf("cot adapter requires a plan writer when projector is configured")
	}
	return &Adapter{
		rawLane:   cfg.RawLane,
		replay:    cfg.Replay,
		projector: cfg.Projector,
		writer:    cfg.Writer,
		clock:     cfg.Clock,
		onEvent:   cfg.OnEvent,
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
	if a.projector != nil {
		plan, err := a.projector.ProjectEvent(event, record.Ref)
		if err != nil {
			a.recordProjectionDrop(err)
			return result, fmt.Errorf("project cot event: %w", err)
		}
		if len(plan.Mutations) == 0 {
			a.recordProjectionDrop(nil)
			if a.onEvent != nil {
				a.onEvent(result)
			}
			return result, nil
		}

		mutations, err := a.writePlan(ctx, event, record.Ref, plan)
		result.Mutations = mutations
		if err != nil {
			a.recordWriteError(err)
			return result, fmt.Errorf("write cot graph plan: %w", err)
		}
		a.recordWrite(now, mutations)
	}
	if a.onEvent != nil {
		a.onEvent(result)
	}
	return result, nil
}

func (a *Adapter) writePlan(
	ctx context.Context,
	event cotcodec.Event,
	sourceRef string,
	plan cotprojector.Plan,
) (int, error) {
	for attempt := 0; attempt < 4; attempt++ {
		if err := a.writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || !a.projector.MarkBornForEvent(event, entityID) {
				return len(plan.Mutations), err
			}
			next, projectErr := a.projector.ProjectEvent(event, sourceRef)
			if projectErr != nil {
				return len(plan.Mutations), fmt.Errorf("reproject cot event after birth reconciliation: %w", projectErr)
			}
			plan = next
			if len(plan.Mutations) == 0 {
				return 0, nil
			}
			continue
		}
		a.projector.MarkBornForPlan(plan)
		return len(plan.Mutations), nil
	}
	return len(plan.Mutations), fmt.Errorf("cot graph birth reconciliation exceeded retry limit")
}

func entityAlreadyExists(err error) (string, bool) {
	var mutationErr *cotprojector.MutationFailureError
	if !errors.As(err, &mutationErr) {
		return "", false
	}
	if mutationErr.Kind != cotprojector.MutationCreate ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists ||
		mutationErr.EntityID == "" {
		return "", false
	}
	return mutationErr.EntityID, true
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

func (a *Adapter) recordWrite(now time.Time, mutations int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = true
	a.health.GraphMutations += uint64(mutations)
	a.health.LastGraphWriteAt = now
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

func (a *Adapter) recordProjectionDrop(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.ProjectionDrops++
	if err != nil {
		a.health.Ready = false
		a.health.LastError = err.Error()
	}
}

func (a *Adapter) recordWriteError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = false
	a.health.WriteErrors++
	a.health.LastError = err.Error()
}
