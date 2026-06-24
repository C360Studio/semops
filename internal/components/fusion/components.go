package fusion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	fusionprojector "github.com/c360studio/semops/internal/projectors/fusion"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

type Subscription interface {
	Unsubscribe() error
}

type Bus interface {
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler func(context.Context, *nats.Msg)) (Subscription, error)
}

type NATSBus struct {
	Client *natsclient.Client
}

func (b NATSBus) Publish(ctx context.Context, subject string, data []byte) error {
	if b.Client == nil {
		return errors.New("fusion NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("fusion NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type ProjectorConfig struct {
	Name             string
	CandidateSubject string
	Registry         *payloadregistry.Registry
	Association      fusionassociation.Config
	Projector        *fusionprojector.Projector
	Writer           PlanWriter
	WriteRetries     int
	WriteTimeout     time.Duration
	Clock            func() time.Time
}

type PlanWriter interface {
	Apply(ctx context.Context, plan fusionprojector.Plan) error
}

type ProjectorComponent struct {
	cfg     ProjectorConfig
	bus     Bus
	decoder *message.Decoder

	mu           sync.Mutex
	state        component.State
	subscription Subscription
	metrics      flowCounters
}

func NewProjectorComponent(cfg ProjectorConfig, bus Bus) (*ProjectorComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-fusion-associate"
	}
	if cfg.CandidateSubject == "" {
		cfg.CandidateSubject = DefaultCandidateSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Projector == nil {
		cfg.Projector = fusionprojector.NewProjector(fusionprojector.Config{})
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("fusion projector component requires a plan writer")
	}
	if cfg.WriteRetries == 0 {
		cfg.WriteRetries = 4
	}
	cfg.Association = normalizeAssociationConfig(cfg.Association)
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("fusion projector component requires a bus")
	}
	return &ProjectorComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *ProjectorComponent) Initialize() error {
	if err := RegisterPayloads(c.cfg.Registry); err != nil {
		return err
	}
	c.decoder = message.NewDecoder(c.cfg.Registry)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	return nil
}

func (c *ProjectorComponent) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	c.mu.Lock()
	if c.state == component.StateStarted {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	sub, err := c.bus.Subscribe(ctx, c.cfg.CandidateSubject, func(msgCtx context.Context, msg *nats.Msg) {
		if err := c.HandleCandidateMessage(msgCtx, msg.Data); err != nil {
			c.recordError(err)
		}
	})
	if err != nil {
		c.markFailed(err)
		return fmt.Errorf("subscribe fusion candidate subject: %w", err)
	}
	c.mu.Lock()
	c.subscription = sub
	c.state = component.StateStarted
	c.metrics.startedAt = c.cfg.Clock().UTC()
	c.mu.Unlock()
	return nil
}

func (c *ProjectorComponent) Stop(time.Duration) error {
	c.mu.Lock()
	sub := c.subscription
	c.subscription = nil
	if c.state == component.StateStarted {
		c.state = component.StateStopped
	}
	c.mu.Unlock()
	if sub != nil {
		return sub.Unsubscribe()
	}
	return nil
}

func (c *ProjectorComponent) HandleCandidateMessage(ctx context.Context, data []byte) error {
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	envelope, err := c.decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("decode fusion candidate BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*CandidateBatchPayload)
	if !ok {
		return fmt.Errorf("fusion projector received payload %T, want *CandidateBatchPayload", envelope.Payload())
	}
	return c.HandleCandidatePayload(ctx, payload)
}

func (c *ProjectorComponent) HandleCandidatePayload(ctx context.Context, payload *CandidateBatchPayload) error {
	primary, candidates, err := payload.TrackObservationsCopy()
	if err != nil {
		return err
	}
	now := c.cfg.Clock().UTC()
	associationCfg := c.cfg.Association
	if associationCfg.ReferenceTime.IsZero() {
		associationCfg.ReferenceTime = now
	}
	evidence := fusionassociation.Associate(primary, candidates, associationCfg)
	if len(evidence) == 0 {
		c.recordMessage(len(primary)+len(candidates), now)
		return nil
	}
	plan, err := c.cfg.Projector.ProjectAssociations(evidence)
	if err != nil {
		return fmt.Errorf("project fusion association evidence: %w", err)
	}
	if len(plan.Mutations) == 0 {
		c.recordMessage(len(primary)+len(candidates), now)
		return nil
	}
	if err := c.writePlan(ctx, evidence, plan); err != nil {
		return err
	}
	c.recordMessage(len(primary)+len(candidates), now)
	return nil
}

func (c *ProjectorComponent) writePlan(
	ctx context.Context,
	evidence []fusionassociation.Evidence,
	plan fusionprojector.Plan,
) error {
	attempts := c.cfg.WriteRetries
	if attempts <= 0 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := c.cfg.Writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || !markBornForEvidence(c.cfg.Projector, evidence, entityID) {
				return fmt.Errorf("write fusion graph plan: %w", err)
			}
			next, projectErr := c.cfg.Projector.ProjectAssociations(evidence)
			if projectErr != nil {
				return fmt.Errorf("reproject fusion associations after birth reconciliation: %w", projectErr)
			}
			plan = next
			if len(plan.Mutations) == 0 {
				return nil
			}
			continue
		}
		c.cfg.Projector.MarkBornForPlan(plan)
		return nil
	}
	return fmt.Errorf("fusion graph birth reconciliation exceeded retry limit")
}

func (c *ProjectorComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "Fusion track-association graph projection processor",
		Version:     "v0.1.0",
	}
}

func (c *ProjectorComponent) InputPorts() []component.Port {
	return []component.Port{
		streamPort("track_candidates", component.DirectionInput, c.cfg.CandidateSubject, CandidateBatchType),
	}
}

func (c *ProjectorComponent) OutputPorts() []component.Port {
	timeout := c.outputTimeout()
	return []component.Port{
		{
			Name:        "graph_create",
			Direction:   component.DirectionOutput,
			Required:    true,
			Description: "SemStreams born-first graph mutation request",
			Config: component.NATSRequestPort{
				Subject: fusionprojector.SubjectEntityCreateWithTriples,
				Timeout: timeout.String(),
				Retries: c.cfg.WriteRetries,
				Interface: &component.InterfaceContract{
					Type:    "graph.CreateEntityWithTriplesRequest",
					Version: "v1",
				},
			},
		},
		{
			Name:        "graph_update",
			Direction:   component.DirectionOutput,
			Required:    true,
			Description: "SemStreams current-state graph mutation request",
			Config: component.NATSRequestPort{
				Subject: fusionprojector.SubjectEntityUpdateWithTriples,
				Timeout: timeout.String(),
				Retries: c.cfg.WriteRetries,
				Interface: &component.InterfaceContract{
					Type:    "graph.UpdateEntityWithTriplesRequest",
					Version: "v1",
				},
			},
		},
	}
}

func (c *ProjectorComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"candidate_subject":   stringProperty("SemStreams subject carrying fusion track-candidate batches", c.cfg.CandidateSubject),
			"owner":               stringProperty("SemStreams projection owner bound through registry/heartbeat", cop.OwnerFusion),
			"write_timeout":       stringProperty("Graph mutation request timeout", c.outputTimeout().String()),
			"max_distance_meters": floatProperty("Maximum source-track distance for association evidence", c.cfg.Association.MaxDistanceMeters),
			"max_time_delta":      stringProperty("Maximum source-track observation time delta", c.cfg.Association.MaxTimeDelta.String()),
			"max_observation_age": stringProperty("Maximum source-track observation age before candidate evidence is filtered", c.cfg.Association.MaxObservationAge.String()),
			"source_priority":     stringProperty("Comma-separated source priority used as equal-score tie-breaker", strings.Join(c.cfg.Association.SourcePriority, ",")),
			"min_confidence":      floatProperty("Minimum association confidence", c.cfg.Association.MinConfidence),
			"ambiguity_margin":    floatProperty("Confidence margin for ambiguous association evidence", c.cfg.Association.AmbiguityMargin),
		},
		Required: []string{"candidate_subject", "owner"},
	}
}

func (c *ProjectorComponent) Health() component.HealthStatus {
	return c.metrics.health(c.state)
}

func (c *ProjectorComponent) DataFlow() component.FlowMetrics {
	return c.metrics.flow()
}

func (c *ProjectorComponent) outputTimeout() time.Duration {
	if c.cfg.WriteTimeout > 0 {
		return c.cfg.WriteTimeout
	}
	return fusionprojector.DefaultWriteTimeout
}

func (c *ProjectorComponent) markFailed(err error) {
	c.mu.Lock()
	c.state = component.StateFailed
	c.mu.Unlock()
	c.recordError(err)
}

func (c *ProjectorComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *ProjectorComponent) recordError(err error) {
	c.metrics.recordError(err)
}

type flowCounters struct {
	mu           sync.Mutex
	startedAt    time.Time
	lastActivity time.Time
	messages     uint64
	bytes        uint64
	errors       int
	lastError    string
}

func (m *flowCounters) recordMessage(size int, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startedAt.IsZero() {
		m.startedAt = now.UTC()
	}
	m.messages++
	m.bytes += uint64(size)
	m.lastActivity = now.UTC()
	m.lastError = ""
}

func (m *flowCounters) recordError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
	m.lastError = err.Error()
	m.lastActivity = time.Now().UTC()
}

func (m *flowCounters) health(state component.State) component.HealthStatus {
	return m.healthAt(state, time.Now().UTC())
}

func (m *flowCounters) healthAt(state component.State, now time.Time) component.HealthStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	healthy := state == component.StateStarted || state == component.StateInitialized
	return component.HealthStatus{
		Healthy:    healthy && m.lastError == "",
		LastCheck:  now.UTC(),
		ErrorCount: m.errors,
		LastError:  m.lastError,
		Uptime:     uptimeSinceAt(m.startedAt, now),
		Status:     state.String(),
	}
}

func (m *flowCounters) flow() component.FlowMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	elapsed := time.Since(m.startedAt).Seconds()
	if m.startedAt.IsZero() || elapsed <= 0 {
		elapsed = 1
	}
	return component.FlowMetrics{
		MessagesPerSecond: float64(m.messages) / elapsed,
		BytesPerSecond:    float64(m.bytes) / elapsed,
		ErrorRate:         float64(m.errors) / elapsed,
		LastActivity:      m.lastActivity,
	}
}

func marshalBaseMessage(
	msgType message.Type,
	payload message.Payload,
	source string,
	observedAt time.Time,
) ([]byte, error) {
	envelope := message.NewBaseMessage(msgType, payload, source, message.WithTime(observedAt.UTC()))
	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal %s BaseMessage: %w", msgType.Key(), err)
	}
	return data, nil
}

func streamPort(name string, direction component.Direction, subject string, msgType message.Type) component.Port {
	return component.Port{
		Name:        name,
		Direction:   direction,
		Required:    true,
		Description: fmt.Sprintf("%s %s", msgType.Key(), name),
		Config: component.NATSPort{
			Subject: subject,
			Interface: &component.InterfaceContract{
				Type:       "message.BaseMessage",
				Version:    "v1",
				Compatible: []string{msgType.Key()},
			},
		},
	}
}

func stringProperty(description, fallback string) component.PropertySchema {
	return component.PropertySchema{Type: "string", Description: description, Default: fallback}
}

func floatProperty(description string, fallback float64) component.PropertySchema {
	return component.PropertySchema{Type: "number", Description: description, Default: strconv.FormatFloat(fallback, 'f', -1, 64)}
}

func normalizeAssociationConfig(cfg fusionassociation.Config) fusionassociation.Config {
	if cfg.MaxDistanceMeters <= 0 {
		cfg.MaxDistanceMeters = fusionassociation.DefaultMaxDistanceMeters
	}
	if cfg.MaxTimeDelta <= 0 {
		cfg.MaxTimeDelta = fusionassociation.DefaultMaxTimeDelta
	}
	if cfg.MaxObservationAge <= 0 {
		cfg.MaxObservationAge = fusionassociation.DefaultMaxObservationAge
	}
	if len(cfg.SourcePriority) == 0 {
		cfg.SourcePriority = append([]string(nil), fusionassociation.DefaultSourcePriority...)
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = fusionassociation.DefaultMinConfidence
	}
	if cfg.AmbiguityMargin <= 0 {
		cfg.AmbiguityMargin = fusionassociation.DefaultAmbiguityMargin
	}
	return cfg
}

func markBornForEvidence(
	projector *fusionprojector.Projector,
	evidence []fusionassociation.Evidence,
	entityID string,
) bool {
	for _, item := range evidence {
		if projector.MarkBornForAssociation(item, entityID) {
			return true
		}
	}
	return false
}

func entityAlreadyExists(err error) (string, bool) {
	var mutationErr *fusionprojector.MutationFailureError
	if !errors.As(err, &mutationErr) {
		return "", false
	}
	if mutationErr.Kind != fusionprojector.MutationCreate ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists ||
		mutationErr.EntityID == "" {
		return "", false
	}
	return mutationErr.EntityID, true
}

func uptimeSinceAt(startedAt time.Time, now time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return now.Sub(startedAt)
}
