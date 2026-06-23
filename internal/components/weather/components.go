package weather

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

	weatherprojector "github.com/c360studio/semops/internal/projectors/weather"
	weathercodec "github.com/c360studio/semops/pkg/adapters/weather"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const DefaultFixturePath = "fixtures/weather/open-meteo-point.json"
const DefaultMaxObservations = 64
const DefaultFreshness = 30 * time.Minute

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
		return errors.New("weather NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("weather NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type FixtureInputConfig struct {
	Name        string
	Source      string
	Provider    string
	QueryShape  string
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
		cfg.Name = "semops-input-weather-fixture"
	}
	if cfg.Source == "" {
		cfg.Source = "weather:fixture"
	}
	if cfg.Provider == "" {
		cfg.Provider = weathercodec.ProviderOpenMeteo
	}
	if cfg.QueryShape == "" {
		cfg.QueryShape = weathercodec.QueryShapePosition
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
		return fmt.Errorf("read weather fixture %q: %w", c.cfg.FixturePath, err)
	}
	fixtureURI, err := fileURI(c.cfg.FixturePath)
	if err != nil {
		return err
	}
	now := c.cfg.Clock().UTC()
	payload := NewRawForecastPayload(
		c.cfg.Source,
		c.cfg.Provider,
		c.cfg.QueryShape,
		c.cfg.FixturePath,
		fixtureURI,
		now,
		raw,
	)
	if err := payload.Validate(); err != nil {
		return err
	}
	data, err := marshalBaseMessage(RawForecastType, payload, c.cfg.Name, now)
	if err != nil {
		return err
	}
	if c.cfg.Bus == nil {
		return errors.New("weather fixture input requires a bus to publish")
	}
	if err := c.cfg.Bus.Publish(ctx, c.cfg.RawSubject, data); err != nil {
		return fmt.Errorf("publish weather raw forecast fixture: %w", err)
	}
	c.state.metrics.recordMessage(len(raw), now)
	return nil
}

func (c *FixtureInputComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "Weather provider-shaped fixture input component",
		Version:     "v0.1.0",
	}
}

func (c *FixtureInputComponent) InputPorts() []component.Port {
	return []component.Port{{
		Name:        "forecast_fixture",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Local provider-shaped tactical weather forecast fixture",
		Config: component.FilePort{
			Path: c.cfg.FixturePath,
		},
	}}
}

func (c *FixtureInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_forecasts", component.DirectionOutput, c.cfg.RawSubject, RawForecastType)}
}

func (c *FixtureInputComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"source":       stringProperty("Source label recorded in raw weather payloads", c.cfg.Source),
			"provider":     stringProperty("Weather provider shape represented by the fixture", c.cfg.Provider),
			"query_shape":  stringProperty("Weather query shape represented by the fixture", c.cfg.QueryShape),
			"fixture_path": stringProperty("Local provider-shaped weather fixture path", c.cfg.FixturePath),
			"raw_subject":  stringProperty("SemStreams subject carrying raw weather payloads", c.cfg.RawSubject),
		},
		Required: []string{"source", "provider", "query_shape", "fixture_path", "raw_subject"},
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
		cfg.Name = "semops-processor-weather-decode"
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
		return nil, errors.New("weather decoder component requires a bus")
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
		return fmt.Errorf("subscribe weather decoder raw subject: %w", err)
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
		return fmt.Errorf("decode weather raw BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*RawForecastPayload)
	if !ok {
		return fmt.Errorf("weather decoder received payload %T, want *RawForecastPayload", envelope.Payload())
	}
	return c.HandleRawPayload(ctx, payload)
}

func (c *DecoderComponent) HandleRawPayload(ctx context.Context, payload *RawForecastPayload) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := payload.Validate(); err != nil {
		return err
	}
	forecast, err := parseForecast(payload)
	if err != nil {
		return err
	}
	decoded := NewDecodedForecastPayload(payload, forecast)
	if err := decoded.Validate(); err != nil {
		return err
	}
	data, err := marshalBaseMessage(DecodedForecastType, decoded, c.cfg.Name, c.cfg.Clock().UTC())
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.DecodedSubject, data); err != nil {
		return fmt.Errorf("publish decoded weather forecast: %w", err)
	}
	c.state.metrics.recordMessage(len(payload.RawJSON), c.cfg.Clock().UTC())
	return nil
}

func (c *DecoderComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "Weather provider-shaped forecast decoder processor",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("raw_forecasts", component.DirectionInput, c.cfg.RawSubject, RawForecastType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{
		streamPort("decoded_forecasts", component.DirectionOutput, c.cfg.DecodedSubject, DecodedForecastType),
	}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_subject":     stringProperty("SemStreams subject carrying raw weather payloads", c.cfg.RawSubject),
			"decoded_subject": stringProperty("SemStreams subject carrying decoded weather payloads", c.cfg.DecodedSubject),
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

type ProjectorConfig struct {
	Name            string
	DecodedSubject  string
	Registry        *payloadregistry.Registry
	Projector       *weatherprojector.Projector
	Writer          PlanWriter
	WriteRetries    int
	WriteTimeout    time.Duration
	Freshness       time.Duration
	MaxObservations int
	Clock           func() time.Time
}

type PlanWriter interface {
	Apply(ctx context.Context, plan weatherprojector.Plan) error
}

type ProjectorComponent struct {
	cfg     ProjectorConfig
	bus     Bus
	state   componentState
	decoder *message.Decoder

	mu           sync.Mutex
	subscription Subscription
}

func NewProjectorComponent(cfg ProjectorConfig, bus Bus) (*ProjectorComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-weather-project"
	}
	if cfg.DecodedSubject == "" {
		cfg.DecodedSubject = DefaultDecodedSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Projector == nil {
		cfg.Projector = weatherprojector.NewProjector(weatherprojector.Config{})
	}
	if cfg.Writer == nil {
		return nil, errors.New("weather projector component requires a plan writer")
	}
	if cfg.WriteRetries == 0 {
		cfg.WriteRetries = 4
	}
	if cfg.Freshness == 0 {
		cfg.Freshness = DefaultFreshness
	}
	if cfg.MaxObservations == 0 {
		cfg.MaxObservations = DefaultMaxObservations
	}
	if cfg.MaxObservations < 0 {
		return nil, errors.New("weather projector max_observations must be non-negative")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, errors.New("weather projector component requires a bus")
	}
	return &ProjectorComponent{cfg: cfg, bus: bus, state: newComponentState(cfg.Clock)}, nil
}

func (c *ProjectorComponent) Initialize() error {
	if err := RegisterPayloads(c.cfg.Registry); err != nil {
		return err
	}
	c.decoder = message.NewDecoder(c.cfg.Registry)
	return c.state.Initialize()
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
	if c.state.Current() == component.StateStarted {
		return nil
	}
	sub, err := c.bus.Subscribe(ctx, c.cfg.DecodedSubject, func(msgCtx context.Context, msg *nats.Msg) {
		if err := c.HandleDecodedMessage(msgCtx, msg.Data); err != nil {
			c.state.metrics.recordError(err)
		}
	})
	if err != nil {
		c.state.Fail(err)
		return fmt.Errorf("subscribe weather projector decoded subject: %w", err)
	}
	c.mu.Lock()
	c.subscription = sub
	c.mu.Unlock()
	return c.state.Start(ctx)
}

func (c *ProjectorComponent) Stop(timeout time.Duration) error {
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

func (c *ProjectorComponent) HandleDecodedMessage(ctx context.Context, data []byte) error {
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	envelope, err := c.decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("decode weather forecast BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*DecodedForecastPayload)
	if !ok {
		return fmt.Errorf("weather projector received payload %T, want *DecodedForecastPayload", envelope.Payload())
	}
	return c.HandleDecodedPayload(ctx, payload)
}

func (c *ProjectorComponent) HandleDecodedPayload(ctx context.Context, payload *DecodedForecastPayload) error {
	if ctx == nil {
		ctx = context.Background()
	}
	forecast, err := payload.ForecastCopy()
	if err != nil {
		return err
	}
	observations, err := weatherprojector.ObservationsFromPointForecast(
		forecast,
		payload.RawRef,
		payload.ReceivedAt,
		c.cfg.Freshness,
	)
	if err != nil {
		return fmt.Errorf("build weather observations: %w", err)
	}
	if c.cfg.MaxObservations > 0 && len(observations) > c.cfg.MaxObservations {
		return fmt.Errorf(
			"weather observation count %d exceeds max_observations %d",
			len(observations),
			c.cfg.MaxObservations,
		)
	}
	plan, err := c.cfg.Projector.ProjectObservations(observations)
	if err != nil {
		return fmt.Errorf("project weather observations: %w", err)
	}
	if len(plan.Mutations) == 0 {
		c.state.metrics.recordMessage(len(payload.RawRef), c.cfg.Clock().UTC())
		return nil
	}
	if err := c.writePlan(ctx, observations, plan); err != nil {
		return err
	}
	c.state.metrics.recordMessage(len(payload.RawRef), c.cfg.Clock().UTC())
	return nil
}

func (c *ProjectorComponent) writePlan(
	ctx context.Context,
	observations []weatherprojector.Observation,
	plan weatherprojector.Plan,
) error {
	attempts := c.cfg.WriteRetries
	if attempts <= 0 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := c.cfg.Writer.Apply(ctx, plan); err != nil {
			entityID, ok := weatherEntityAlreadyExists(err)
			if !ok || !c.markBornForObservations(observations, entityID) {
				return fmt.Errorf("write weather graph plan: %w", err)
			}
			next, projectErr := c.cfg.Projector.ProjectObservations(observations)
			if projectErr != nil {
				return fmt.Errorf("reproject weather observations after birth reconciliation: %w", projectErr)
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
	return fmt.Errorf("weather graph birth reconciliation exceeded retry limit")
}

func (c *ProjectorComponent) markBornForObservations(
	observations []weatherprojector.Observation,
	entityID string,
) bool {
	for _, observation := range observations {
		if c.cfg.Projector.MarkBornForObservation(observation, entityID) {
			return true
		}
	}
	return false
}

func (c *ProjectorComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "Weather governed graph projection processor",
		Version:     "v0.1.0",
	}
}

func (c *ProjectorComponent) InputPorts() []component.Port {
	return []component.Port{
		streamPort("decoded_forecasts", component.DirectionInput, c.cfg.DecodedSubject, DecodedForecastType),
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
				Subject: weatherprojector.SubjectEntityCreateWithTriples,
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
			Description: "SemStreams append-evidence graph mutation request",
			Config: component.NATSRequestPort{
				Subject: weatherprojector.SubjectEntityUpdateWithTriples,
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
			"decoded_subject":  stringProperty("SemStreams subject carrying decoded weather forecasts", c.cfg.DecodedSubject),
			"owner":            stringProperty("SemStreams projection owner bound through registry/heartbeat", cop.OwnerWeather),
			"write_timeout":    stringProperty("Graph mutation request timeout", c.outputTimeout().String()),
			"freshness":        stringProperty("Weather observation freshness window", c.cfg.Freshness.String()),
			"max_observations": intProperty("Maximum observations accepted per decoded forecast payload", c.cfg.MaxObservations),
		},
		Required: []string{"decoded_subject", "owner", "max_observations"},
	}
}

func (c *ProjectorComponent) Health() component.HealthStatus {
	return c.state.Health()
}

func (c *ProjectorComponent) DataFlow() component.FlowMetrics {
	return c.state.DataFlow()
}

func (c *ProjectorComponent) outputTimeout() time.Duration {
	if c.cfg.WriteTimeout > 0 {
		return c.cfg.WriteTimeout
	}
	return weatherprojector.DefaultWriteTimeout
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

func intProperty(description string, fallback int) component.PropertySchema {
	return component.PropertySchema{Type: "int", Description: description, Default: fallback}
}

func weatherEntityAlreadyExists(err error) (string, bool) {
	var mutationErr *weatherprojector.MutationFailureError
	if !errors.As(err, &mutationErr) ||
		mutationErr.Kind != weatherprojector.MutationCreate ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists ||
		mutationErr.EntityID == "" {
		return "", false
	}
	return mutationErr.EntityID, true
}

func fileURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve weather fixture path %q: %w", path, err)
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String(), nil
}

func parseForecast(payload *RawForecastPayload) (weathercodec.PointForecast, error) {
	switch {
	case payload.Provider == weathercodec.ProviderOpenMeteo &&
		payload.QueryShape == weathercodec.QueryShapePosition:
		forecast, err := weathercodec.ParseOpenMeteoPointForecast(payload.RawJSON)
		if err != nil {
			return weathercodec.PointForecast{}, fmt.Errorf("parse Open-Meteo point forecast: %w", err)
		}
		return forecast, nil
	case payload.Provider == weathercodec.ProviderOGCEDR &&
		payload.QueryShape == weathercodec.QueryShapePosition:
		forecast, err := weathercodec.ParseOGCEDRPositionForecast(payload.RawJSON)
		if err != nil {
			return weathercodec.PointForecast{}, fmt.Errorf("parse OGC EDR position forecast: %w", err)
		}
		return forecast, nil
	default:
		return weathercodec.PointForecast{}, fmt.Errorf(
			"weather decoder supports %s/%s or %s/%s, got %s/%s",
			weathercodec.ProviderOpenMeteo,
			weathercodec.QueryShapePosition,
			weathercodec.ProviderOGCEDR,
			weathercodec.QueryShapePosition,
			payload.Provider,
			payload.QueryShape,
		)
	}
}

func uptimeSinceAt(startedAt time.Time, now time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return now.Sub(startedAt)
}
