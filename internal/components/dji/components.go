package dji

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	djicodec "github.com/c360studio/semops/pkg/adapters/dji"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const DefaultFixturePath = "fixtures/dji/telemetry-media.json"

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
		return errors.New("dji NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("dji NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type FixtureInputConfig struct {
	Name        string
	Source      string
	FixturePath string
	RawSubject  string
	Registry    *payloadregistry.Registry
	Bus         Bus
	Clock       func() time.Time
}

type FixtureInputComponent struct {
	cfg   FixtureInputConfig
	state componentState
}

func NewFixtureInputComponent(cfg FixtureInputConfig) (*FixtureInputComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-input-dji-fixture"
	}
	if cfg.Source == "" {
		cfg.Source = "dji:fixture"
	}
	if cfg.FixturePath == "" {
		cfg.FixturePath = DefaultFixturePath
	}
	if cfg.RawSubject == "" {
		cfg.RawSubject = DefaultRawSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &FixtureInputComponent{cfg: cfg, state: newComponentState(cfg.Clock)}, nil
}

func (c *FixtureInputComponent) Initialize() error {
	if err := RegisterPayloads(c.cfg.Registry); err != nil {
		return err
	}
	return c.state.Initialize()
}

func (c *FixtureInputComponent) Start(ctx context.Context) error {
	alreadyStarted := c.state.Current() == component.StateStarted
	if err := c.state.Start(ctx); err != nil {
		return err
	}
	if alreadyStarted || c.cfg.Bus == nil {
		return nil
	}
	if err := c.PublishOnce(ctx); err != nil {
		c.state.metrics.recordError(err)
		return err
	}
	return nil
}

func (c *FixtureInputComponent) Stop(timeout time.Duration) error {
	return c.state.Stop(timeout)
}

func (c *FixtureInputComponent) PublishOnce(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := os.ReadFile(c.cfg.FixturePath)
	if err != nil {
		return fmt.Errorf("read DJI telemetry fixture %q: %w", c.cfg.FixturePath, err)
	}
	fixtureURI, err := fileURI(c.cfg.FixturePath)
	if err != nil {
		return err
	}
	now := c.cfg.Clock().UTC()
	payload := NewRawTelemetryPayload(c.cfg.Source, c.cfg.FixturePath, fixtureURI, now, raw)
	if err := payload.Validate(); err != nil {
		return err
	}
	data, err := marshalBaseMessage(RawTelemetryType, payload, c.cfg.Name, now)
	if err != nil {
		return err
	}
	if c.cfg.Bus == nil {
		return errors.New("dji fixture input requires a bus to publish")
	}
	if err := c.cfg.Bus.Publish(ctx, c.cfg.RawSubject, data); err != nil {
		return fmt.Errorf("publish DJI raw telemetry fixture: %w", err)
	}
	c.state.metrics.recordMessage(len(raw), now)
	return nil
}

func (c *FixtureInputComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "DJI synthetic fixture telemetry input component",
		Version:     "v0.1.0",
	}
}

func (c *FixtureInputComponent) InputPorts() []component.Port {
	return []component.Port{{
		Name:        "telemetry_fixture",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Local synthetic DJI-shaped telemetry fixture",
		Config: component.FilePort{
			Path: c.cfg.FixturePath,
		},
	}}
}

func (c *FixtureInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_telemetry", component.DirectionOutput, c.cfg.RawSubject, RawTelemetryType)}
}

func (c *FixtureInputComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"source":       stringProperty("Source label recorded in raw DJI telemetry payloads", c.cfg.Source),
			"fixture_path": stringProperty("Local synthetic DJI-shaped telemetry fixture path", c.cfg.FixturePath),
			"raw_subject":  stringProperty("SemStreams subject carrying raw DJI telemetry payloads", c.cfg.RawSubject),
		},
		Required: []string{"source", "fixture_path", "raw_subject"},
	}
}

func (c *FixtureInputComponent) Health() component.HealthStatus {
	return c.state.Health()
}

func (c *FixtureInputComponent) DataFlow() component.FlowMetrics {
	return c.state.DataFlow()
}

type DecoderConfig struct {
	Name           string
	RawSubject     string
	DecodedSubject string
	Registry       *payloadregistry.Registry
	Clock          func() time.Time
}

type DecoderComponent struct {
	cfg     DecoderConfig
	bus     Bus
	state   componentState
	decoder *message.Decoder

	mu           sync.Mutex
	subscription Subscription
}

func NewDecoderComponent(cfg DecoderConfig, bus Bus) (*DecoderComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-dji-decode"
	}
	if cfg.RawSubject == "" {
		cfg.RawSubject = DefaultRawSubject
	}
	if cfg.DecodedSubject == "" {
		cfg.DecodedSubject = DefaultDecodedSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, errors.New("dji decoder component requires a bus")
	}
	return &DecoderComponent{cfg: cfg, bus: bus, state: newComponentState(cfg.Clock)}, nil
}

func (c *DecoderComponent) Initialize() error {
	if err := RegisterPayloads(c.cfg.Registry); err != nil {
		return err
	}
	c.decoder = message.NewDecoder(c.cfg.Registry)
	return c.state.Initialize()
}

func (c *DecoderComponent) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	if c.state.Current() == component.StateStarted {
		return nil
	}
	sub, err := c.bus.Subscribe(ctx, c.cfg.RawSubject, func(msgCtx context.Context, msg *nats.Msg) {
		if err := c.HandleRawMessage(msgCtx, msg.Data); err != nil {
			c.state.metrics.recordError(err)
		}
	})
	if err != nil {
		c.state.Fail(err)
		return fmt.Errorf("subscribe DJI decoder raw subject: %w", err)
	}
	c.mu.Lock()
	c.subscription = sub
	c.mu.Unlock()
	return c.state.Start(ctx)
}

func (c *DecoderComponent) Stop(timeout time.Duration) error {
	c.mu.Lock()
	sub := c.subscription
	c.subscription = nil
	c.mu.Unlock()
	if sub != nil {
		if err := sub.Unsubscribe(); err != nil {
			c.state.metrics.recordError(err)
			return err
		}
	}
	return c.state.Stop(timeout)
}

func (c *DecoderComponent) HandleRawMessage(ctx context.Context, data []byte) error {
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	envelope, err := c.decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("decode DJI raw BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*RawTelemetryPayload)
	if !ok {
		return fmt.Errorf("DJI decoder received payload %T, want *RawTelemetryPayload", envelope.Payload())
	}
	return c.HandleRawPayload(ctx, payload)
}

func (c *DecoderComponent) HandleRawPayload(ctx context.Context, payload *RawTelemetryPayload) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := payload.Validate(); err != nil {
		return err
	}
	record, err := djicodec.ParseTelemetryRecord(payload.RawJSON)
	if err != nil {
		return fmt.Errorf("parse DJI telemetry fixture: %w", err)
	}
	decoded := NewDecodedTelemetryPayload(payload, record)
	if err := decoded.Validate(); err != nil {
		return err
	}
	data, err := marshalBaseMessage(DecodedTelemetryType, decoded, c.cfg.Name, record.ObservedAt)
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.DecodedSubject, data); err != nil {
		return fmt.Errorf("publish decoded DJI telemetry: %w", err)
	}
	c.state.metrics.recordMessage(len(payload.RawJSON), c.cfg.Clock().UTC())
	return nil
}

func (c *DecoderComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "DJI synthetic telemetry decoder processor",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("raw_telemetry", component.DirectionInput, c.cfg.RawSubject, RawTelemetryType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{
		streamPort("decoded_telemetry", component.DirectionOutput, c.cfg.DecodedSubject, DecodedTelemetryType),
	}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_subject":     stringProperty("SemStreams subject carrying raw DJI telemetry payloads", c.cfg.RawSubject),
			"decoded_subject": stringProperty("SemStreams subject carrying decoded DJI telemetry payloads", c.cfg.DecodedSubject),
		},
		Required: []string{"raw_subject", "decoded_subject"},
	}
}

func (c *DecoderComponent) Health() component.HealthStatus {
	return c.state.Health()
}

func (c *DecoderComponent) DataFlow() component.FlowMetrics {
	return c.state.DataFlow()
}

type componentState struct {
	mu      sync.Mutex
	state   component.State
	clock   func() time.Time
	metrics flowCounters
}

func newComponentState(clock func() time.Time) componentState {
	if clock == nil {
		clock = time.Now
	}
	return componentState{state: component.StateCreated, clock: clock}
}

func (s *componentState) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == component.StateCreated {
		s.state = component.StateInitialized
	}
	return nil
}

func (s *componentState) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == component.StateStarted {
		return nil
	}
	if s.state == component.StateCreated {
		s.state = component.StateInitialized
	}
	s.state = component.StateStarted
	s.metrics.startedAt = s.clock().UTC()
	return nil
}

func (s *componentState) Stop(time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == component.StateStarted {
		s.state = component.StateStopped
	}
	return nil
}

func (s *componentState) Current() component.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *componentState) Fail(err error) {
	s.mu.Lock()
	s.state = component.StateFailed
	s.mu.Unlock()
	s.metrics.recordError(err)
}

func (s *componentState) Health() component.HealthStatus {
	return s.metrics.health(s.Current())
}

func (s *componentState) DataFlow() component.FlowMetrics {
	return s.metrics.flow()
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
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	healthy := state == component.StateStarted || state == component.StateInitialized
	return component.HealthStatus{
		Healthy:    healthy && m.lastError == "",
		LastCheck:  now,
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

func stringProperty(description, fallback string) component.PropertySchema {
	return component.PropertySchema{Type: "string", Description: description, Default: fallback}
}

func fileURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve DJI fixture path %q: %w", path, err)
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String(), nil
}

func uptimeSinceAt(startedAt time.Time, now time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return now.Sub(startedAt)
}
