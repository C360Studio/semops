package adsb

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semstreams/graph"
)

type ReplayAppender interface {
	Append(record adsbcodec.RawSnapshotRecord) error
}

type PlanWriter interface {
	Apply(ctx context.Context, plan adsbprojector.Plan) error
}

type Config struct {
	Source     string
	RawLane    *adsbcodec.RawLane
	Replay     ReplayAppender
	Projector  *adsbprojector.Projector
	Writer     PlanWriter
	Clock      func() time.Time
	OnSnapshot func(IngestResult)
}

type Adapter struct {
	rawLane    *adsbcodec.RawLane
	replay     ReplayAppender
	projector  *adsbprojector.Projector
	writer     PlanWriter
	clock      func() time.Time
	onSnapshot func(IngestResult)

	mu     sync.RWMutex
	health Health
}

type IngestResult struct {
	RawRef        string
	Snapshot      adsbcodec.OpenSkySnapshot
	StatesDecoded int
	Mutations     int
}

type Health struct {
	Source            string
	Ready             bool
	SnapshotsReceived uint64
	SnapshotsCaptured uint64
	SnapshotsDecoded  uint64
	StatesDecoded     uint64
	GraphMutations    uint64
	ParseErrors       uint64
	CaptureErrors     uint64
	ReplayErrors      uint64
	ProjectionDrops   uint64
	WriteErrors       uint64
	LastSnapshotAt    time.Time
	LastGraphWriteAt  time.Time
	LastRawRef        string
	LastICAO24        string
	LastError         string
}

func NewAdapter(cfg Config) (*Adapter, error) {
	if cfg.Source == "" {
		cfg.Source = "adsb"
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.RawLane == nil {
		cfg.RawLane = adsbcodec.NewRawLane(adsbcodec.RawLaneConfig{Source: cfg.Source, Clock: cfg.Clock})
	}
	if cfg.Projector != nil && cfg.Writer == nil {
		return nil, fmt.Errorf("adsb adapter requires a plan writer when projector is configured")
	}
	return &Adapter{
		rawLane:    cfg.RawLane,
		replay:     cfg.Replay,
		projector:  cfg.Projector,
		writer:     cfg.Writer,
		clock:      cfg.Clock,
		onSnapshot: cfg.OnSnapshot,
		health: Health{
			Source: cfg.Source,
			Ready:  true,
		},
	}, nil
}

func (a *Adapter) IngestSnapshot(ctx context.Context, raw []byte) (IngestResult, error) {
	if a == nil {
		return IngestResult{}, fmt.Errorf("adsb adapter is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return IngestResult{}, err
	}

	now := a.clock().UTC()
	a.recordReceived(now)

	snapshot, parseErr := adsbcodec.ParseOpenSkySnapshot(raw)
	if parseErr != nil {
		record, captureErr := a.rawLane.Capture(raw, nil)
		if captureErr != nil {
			a.recordCaptureError(captureErr)
			return IngestResult{}, fmt.Errorf("capture unparsable adsb snapshot: %w", captureErr)
		}
		if err := a.appendReplay(record); err != nil {
			a.recordReplayError(err)
			return IngestResult{RawRef: record.Ref}, err
		}
		err := fmt.Errorf("parse adsb snapshot: %w", parseErr)
		a.recordParseError(err, record.Ref)
		return IngestResult{RawRef: record.Ref}, err
	}

	record, err := a.rawLane.Capture(raw, &snapshot)
	if err != nil {
		a.recordCaptureError(err)
		return IngestResult{}, fmt.Errorf("capture adsb snapshot: %w", err)
	}
	if err := a.appendReplay(record); err != nil {
		a.recordReplayError(err)
		return IngestResult{RawRef: record.Ref, Snapshot: snapshot}, err
	}

	result := IngestResult{
		RawRef:        record.Ref,
		Snapshot:      snapshot,
		StatesDecoded: len(snapshot.States),
	}
	a.recordDecoded(record.Ref, snapshot)
	if a.projector != nil {
		sources := sourceStates(snapshot.States, record.Ref)
		plan, err := a.projector.ProjectStates(sources)
		if err != nil {
			a.recordProjectionDrop(err)
			return result, fmt.Errorf("project adsb snapshot: %w", err)
		}
		if len(plan.Mutations) == 0 {
			a.recordProjectionDrop(nil)
			if a.onSnapshot != nil {
				a.onSnapshot(result)
			}
			return result, nil
		}

		mutations, err := a.writePlan(ctx, snapshot.States, sources, plan)
		result.Mutations = mutations
		if err != nil {
			a.recordWriteError(err)
			return result, fmt.Errorf("write adsb graph plan: %w", err)
		}
		a.recordWrite(now, mutations)
	}
	if a.onSnapshot != nil {
		a.onSnapshot(result)
	}
	return result, nil
}

func (a *Adapter) writePlan(
	ctx context.Context,
	states []adsbcodec.StateVector,
	sources []adsbprojector.SourceState,
	plan adsbprojector.Plan,
) (int, error) {
	for attempt := 0; attempt < 4; attempt++ {
		if err := a.writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || !a.markBornForEntity(states, entityID) {
				return len(plan.Mutations), err
			}
			next, projectErr := a.projector.ProjectStates(sources)
			if projectErr != nil {
				return len(plan.Mutations), fmt.Errorf("reproject adsb snapshot after birth reconciliation: %w", projectErr)
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
	return len(plan.Mutations), fmt.Errorf("adsb graph birth reconciliation exceeded retry limit")
}

func entityAlreadyExists(err error) (string, bool) {
	var mutationErr *adsbprojector.MutationFailureError
	if !errors.As(err, &mutationErr) {
		return "", false
	}
	if mutationErr.Kind != adsbprojector.MutationCreate ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists ||
		mutationErr.EntityID == "" {
		return "", false
	}
	return mutationErr.EntityID, true
}

func (a *Adapter) markBornForEntity(states []adsbcodec.StateVector, entityID string) bool {
	for _, state := range states {
		if a.projector.MarkBornForState(state, entityID) {
			return true
		}
	}
	return false
}

func (a *Adapter) Health() Health {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.health
}

func (a *Adapter) RawLane() *adsbcodec.RawLane {
	return a.rawLane
}

func (a *Adapter) appendReplay(record adsbcodec.RawSnapshotRecord) error {
	if a.replay == nil {
		return nil
	}
	if err := a.replay.Append(record); err != nil {
		return fmt.Errorf("append adsb replay record %q: %w", record.Ref, err)
	}
	return nil
}

func (a *Adapter) recordReceived(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.SnapshotsReceived++
	a.health.LastSnapshotAt = now
}

func (a *Adapter) recordDecoded(rawRef string, snapshot adsbcodec.OpenSkySnapshot) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = true
	a.health.SnapshotsCaptured++
	a.health.SnapshotsDecoded++
	a.health.StatesDecoded += uint64(len(snapshot.States))
	a.health.LastRawRef = rawRef
	a.health.LastSnapshotAt = snapshot.Time
	a.health.LastICAO24 = lastICAO24(snapshot.States)
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
	a.health.SnapshotsCaptured++
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

func sourceStates(states []adsbcodec.StateVector, sourceRef string) []adsbprojector.SourceState {
	out := make([]adsbprojector.SourceState, 0, len(states))
	for _, state := range states {
		out = append(out, adsbprojector.SourceState{State: state, SourceRef: sourceRef})
	}
	return out
}

func lastICAO24(states []adsbcodec.StateVector) string {
	if len(states) == 0 {
		return ""
	}
	return states[len(states)-1].ICAO24
}
