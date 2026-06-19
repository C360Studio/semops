package mavlink

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semstreams/graph"
)

type PlanWriter interface {
	Apply(ctx context.Context, plan mavprojector.Plan) error
}

type Config struct {
	Source    string
	Parser    *mavcodec.Parser
	RawLane   *mavcodec.RawLane
	Projector *mavprojector.Projector
	Writer    PlanWriter
	Clock     func() time.Time
}

type Adapter struct {
	parser    *mavcodec.Parser
	rawLane   *mavcodec.RawLane
	projector *mavprojector.Projector
	writer    PlanWriter
	clock     func() time.Time

	mu     sync.RWMutex
	health Health
}

type IngestResult struct {
	RawRef         string
	MessageID      uint32
	PacketsDecoded int
	Mutations      int
}

type Health struct {
	Source           string
	Ready            bool
	FramesReceived   uint64
	FramesCaptured   uint64
	PacketsDecoded   uint64
	GraphMutations   uint64
	ParseErrors      uint64
	CaptureErrors    uint64
	ProjectionDrops  uint64
	WriteErrors      uint64
	LastFrameAt      time.Time
	LastGraphWriteAt time.Time
	LastRawRef       string
	LastError        string
}

func NewAdapter(cfg Config) (*Adapter, error) {
	if cfg.Source == "" {
		cfg.Source = "mavlink"
	}
	if cfg.Parser == nil {
		cfg.Parser = mavcodec.NewParser()
	}
	if cfg.RawLane == nil {
		cfg.RawLane = mavcodec.NewRawLane(mavcodec.RawLaneConfig{Source: cfg.Source})
	}
	if cfg.Projector == nil {
		cfg.Projector = mavprojector.NewProjector(mavprojector.Config{})
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("mavlink adapter requires a plan writer")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}

	adapter := &Adapter{
		parser:    cfg.Parser,
		rawLane:   cfg.RawLane,
		projector: cfg.Projector,
		writer:    cfg.Writer,
		clock:     cfg.Clock,
		health: Health{
			Source: cfg.Source,
			Ready:  true,
		},
	}
	return adapter, nil
}

func (a *Adapter) IngestFrame(ctx context.Context, frame []byte) (IngestResult, error) {
	if a == nil {
		return IngestResult{}, fmt.Errorf("mavlink adapter is nil")
	}
	now := a.clock().UTC()
	a.recordFrame(now)

	packets, parseErr := a.parser.Parse(frame)
	if parseErr != nil {
		record, captureErr := a.rawLane.Capture(frame, nil)
		if captureErr != nil {
			a.recordCaptureError(captureErr)
			return IngestResult{}, fmt.Errorf("capture unparsable mavlink frame: %w", captureErr)
		}
		err := fmt.Errorf("parse mavlink frame: %w", parseErr)
		a.recordParseError(err, record.Ref)
		return IngestResult{RawRef: record.Ref}, err
	}
	if len(packets) != 1 {
		record, captureErr := a.rawLane.Capture(frame, nil)
		if captureErr != nil {
			a.recordCaptureError(captureErr)
			return IngestResult{}, fmt.Errorf("capture invalid mavlink frame: %w", captureErr)
		}
		err := fmt.Errorf("expected exactly one valid MAVLink packet, got %d", len(packets))
		a.recordParseError(err, record.Ref)
		return IngestResult{RawRef: record.Ref}, err
	}

	packet := packets[0]
	record, err := a.rawLane.Capture(frame, packet)
	if err != nil {
		a.recordCaptureError(err)
		return IngestResult{}, fmt.Errorf("capture mavlink frame: %w", err)
	}
	result := IngestResult{
		RawRef:         record.Ref,
		MessageID:      packet.MessageID,
		PacketsDecoded: 1,
	}
	a.recordPacket(record.Ref)

	plan, err := a.projector.ProjectPacket(packet)
	if err != nil {
		a.recordProjectionDrop(err)
		return result, fmt.Errorf("project mavlink packet: %w", err)
	}
	if len(plan.Mutations) == 0 {
		a.recordProjectionDrop(nil)
		return result, nil
	}

	mutations, err := a.writePlan(ctx, packet, plan)
	result.Mutations = mutations
	if err != nil {
		a.recordWriteError(err)
		return result, fmt.Errorf("write mavlink graph plan: %w", err)
	}
	a.recordWrite(now, mutations)
	return result, nil
}

func (a *Adapter) writePlan(ctx context.Context, packet *mavcodec.Packet, plan mavprojector.Plan) (int, error) {
	for attempt := 0; attempt < 4; attempt++ {
		if err := a.writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || !a.projector.MarkBornForPacket(packet, entityID) {
				return len(plan.Mutations), err
			}
			next, projectErr := a.projector.ProjectPacket(packet)
			if projectErr != nil {
				return len(plan.Mutations), fmt.Errorf("reproject mavlink packet after birth reconciliation: %w", projectErr)
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
	return len(plan.Mutations), fmt.Errorf("mavlink graph birth reconciliation exceeded retry limit")
}

func entityAlreadyExists(err error) (string, bool) {
	var mutationErr *mavprojector.MutationFailureError
	if !errors.As(err, &mutationErr) {
		return "", false
	}
	if mutationErr.Kind != mavprojector.MutationCreate ||
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

func (a *Adapter) RawLane() *mavcodec.RawLane {
	return a.rawLane
}

func (a *Adapter) recordFrame(now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.FramesReceived++
	a.health.LastFrameAt = now
}

func (a *Adapter) recordPacket(rawRef string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.health.Ready = true
	a.health.FramesCaptured++
	a.health.PacketsDecoded++
	a.health.LastRawRef = rawRef
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
	a.health.FramesCaptured++
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
