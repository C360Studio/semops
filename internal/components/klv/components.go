package klv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const (
	DefaultMediaPath       = "fixtures/klv"
	DefaultMediaPattern    = "*.ts"
	DefaultOwner           = "semops.feed.klv"
	DefaultWriteTimeout    = 5 * time.Second
	DefaultMaxPacketBytes  = 64 * 1024
	DefaultSupportedSubset = "misb-st-0601-platform-sensor-frame"

	SubjectEntityCreateWithTriples = "graph.mutation.entity.create_with_triples"
	SubjectEntityUpdateWithTriples = "graph.mutation.entity.update_with_triples"
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
		return errors.New("klv NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("klv NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type MediaRefInputConfig struct {
	Name            string
	Source          string
	MediaPath       string
	MediaPattern    string
	MediaRefSubject string
	Clock           func() time.Time
}

type MediaRefInputComponent struct {
	cfg   MediaRefInputConfig
	state componentState
}

func NewMediaRefInputComponent(cfg MediaRefInputConfig) (*MediaRefInputComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-input-klv-media-ref"
	}
	if cfg.Source == "" {
		cfg.Source = "klv:media-ref"
	}
	if cfg.MediaPath == "" {
		cfg.MediaPath = DefaultMediaPath
	}
	if cfg.MediaPattern == "" {
		cfg.MediaPattern = DefaultMediaPattern
	}
	if cfg.MediaRefSubject == "" {
		cfg.MediaRefSubject = DefaultMediaRefSubject
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &MediaRefInputComponent{cfg: cfg, state: newComponentState(cfg.Clock)}, nil
}

func (c *MediaRefInputComponent) Initialize() error {
	return c.state.Initialize()
}

func (c *MediaRefInputComponent) Start(ctx context.Context) error {
	return c.state.Start(ctx)
}

func (c *MediaRefInputComponent) Stop(timeout time.Duration) error {
	return c.state.Stop(timeout)
}

func (c *MediaRefInputComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "KLV media-reference input component",
		Version:     "v0.1.0",
	}
}

func (c *MediaRefInputComponent) InputPorts() []component.Port {
	return []component.Port{{
		Name:        "media_files",
		Direction:   component.DirectionInput,
		Required:    false,
		Description: "Local video-plus-KLV fixture files or mounted media directory",
		Config: component.FilePort{
			Path:    c.cfg.MediaPath,
			Pattern: c.cfg.MediaPattern,
		},
	}}
}

func (c *MediaRefInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("media_refs", component.DirectionOutput, c.cfg.MediaRefSubject, MediaRefType)}
}

func (c *MediaRefInputComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"source":            stringProperty("Source label recorded in KLV media-reference payloads", c.cfg.Source),
			"media_path":        stringProperty("Local fixture/media path for KLV media-reference input", c.cfg.MediaPath),
			"media_pattern":     stringProperty("File glob used for local KLV fixture/media discovery", c.cfg.MediaPattern),
			"media_ref_subject": stringProperty("SemStreams subject carrying KLV media-reference payloads", c.cfg.MediaRefSubject),
		},
		Required: []string{"source", "media_ref_subject"},
	}
}

func (c *MediaRefInputComponent) Health() component.HealthStatus {
	return c.state.Health()
}

func (c *MediaRefInputComponent) DataFlow() component.FlowMetrics {
	return c.state.DataFlow()
}

type DemuxConfig struct {
	Name            string
	MediaRefSubject string
	PacketSubject   string
	MaxPacketBytes  int
	Clock           func() time.Time
}

type DemuxComponent struct {
	cfg   DemuxConfig
	state componentState
}

func NewDemuxComponent(cfg DemuxConfig) (*DemuxComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-klv-demux"
	}
	if cfg.MediaRefSubject == "" {
		cfg.MediaRefSubject = DefaultMediaRefSubject
	}
	if cfg.PacketSubject == "" {
		cfg.PacketSubject = DefaultPacketSubject
	}
	if cfg.MaxPacketBytes == 0 {
		cfg.MaxPacketBytes = DefaultMaxPacketBytes
	}
	if cfg.MaxPacketBytes < 0 {
		return nil, fmt.Errorf("KLV demux max_packet_bytes must be greater than zero")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &DemuxComponent{cfg: cfg, state: newComponentState(cfg.Clock)}, nil
}

func (c *DemuxComponent) Initialize() error {
	return c.state.Initialize()
}

func (c *DemuxComponent) Start(ctx context.Context) error {
	return c.state.Start(ctx)
}

func (c *DemuxComponent) Stop(timeout time.Duration) error {
	return c.state.Stop(timeout)
}

func (c *DemuxComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "KLV MPEG-TS demux processor component",
		Version:     "v0.1.0",
	}
}

func (c *DemuxComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("media_refs", component.DirectionInput, c.cfg.MediaRefSubject, MediaRefType)}
}

func (c *DemuxComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("klv_packets", component.DirectionOutput, c.cfg.PacketSubject, PacketType)}
}

func (c *DemuxComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"media_ref_subject": stringProperty("SemStreams subject carrying KLV media-reference payloads", c.cfg.MediaRefSubject),
			"packet_subject":    stringProperty("SemStreams subject carrying demuxed KLV packet payloads", c.cfg.PacketSubject),
			"max_packet_bytes":  intProperty("Maximum bounded KLV packet bytes carried in stream payloads", c.cfg.MaxPacketBytes),
		},
		Required: []string{"media_ref_subject", "packet_subject", "max_packet_bytes"},
	}
}

func (c *DemuxComponent) Health() component.HealthStatus {
	return c.state.Health()
}

func (c *DemuxComponent) DataFlow() component.FlowMetrics {
	return c.state.DataFlow()
}

type DecoderConfig struct {
	Name            string
	Source          string
	PacketSubject   string
	FrameSubject    string
	SupportedSubset string
	Registry        *payloadregistry.Registry
	Bus             Bus
	Clock           func() time.Time
}

type DecoderComponent struct {
	cfg     DecoderConfig
	state   componentState
	decoder *message.Decoder

	mu           sync.Mutex
	subscription Subscription
}

func NewDecoderComponent(cfg DecoderConfig) (*DecoderComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-klv-decode"
	}
	if cfg.Source == "" {
		cfg.Source = DefaultDecodeSource
	}
	if cfg.PacketSubject == "" {
		cfg.PacketSubject = DefaultPacketSubject
	}
	if cfg.FrameSubject == "" {
		cfg.FrameSubject = DefaultFrameSubject
	}
	if cfg.SupportedSubset == "" {
		cfg.SupportedSubset = DefaultSupportedSubset
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &DecoderComponent{cfg: cfg, state: newComponentState(cfg.Clock)}, nil
}

func (c *DecoderComponent) Initialize() error {
	if err := RegisterPayloads(c.cfg.Registry); err != nil {
		return err
	}
	c.decoder = message.NewDecoder(c.cfg.Registry)
	return c.state.Initialize()
}

func (c *DecoderComponent) Start(ctx context.Context) error {
	if c.cfg.Bus == nil {
		return c.state.Start(ctx)
	}
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	if c.state.Current() == component.StateStarted {
		return nil
	}
	sub, err := c.cfg.Bus.Subscribe(ctx, c.cfg.PacketSubject, func(msgCtx context.Context, msg *nats.Msg) {
		if err := c.HandlePacketMessage(msgCtx, msg.Data); err != nil {
			c.state.metrics.recordError(err)
		}
	})
	if err != nil {
		c.state.metrics.recordError(err)
		return fmt.Errorf("subscribe KLV decoder packet subject: %w", err)
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

func (c *DecoderComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "MISB ST 0601 KLV decode processor component",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("klv_packets", component.DirectionInput, c.cfg.PacketSubject, PacketType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("misb0601_frames", component.DirectionOutput, c.cfg.FrameSubject, MISB0601FrameType)}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"packet_subject":   stringProperty("SemStreams subject carrying demuxed KLV packet payloads", c.cfg.PacketSubject),
			"frame_subject":    stringProperty("SemStreams subject carrying decoded MISB ST 0601 frame payloads", c.cfg.FrameSubject),
			"source":           stringProperty("Source label recorded in decoded MISB ST 0601 frame payloads", c.cfg.Source),
			"supported_subset": stringProperty("Named MISB ST 0601 field subset for the first decoder slice", c.cfg.SupportedSubset),
		},
		Required: []string{"packet_subject", "frame_subject", "source", "supported_subset"},
	}
}

func (c *DecoderComponent) Health() component.HealthStatus {
	return c.state.Health()
}

func (c *DecoderComponent) DataFlow() component.FlowMetrics {
	return c.state.DataFlow()
}

func (c *DecoderComponent) HandlePacketMessage(ctx context.Context, data []byte) error {
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	envelope, err := c.decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("decode KLV packet BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*PacketPayload)
	if !ok {
		return fmt.Errorf("KLV decoder received payload %T, want *PacketPayload", envelope.Payload())
	}
	return c.HandlePacketPayload(ctx, payload)
}

func (c *DecoderComponent) HandlePacketPayload(ctx context.Context, payload *PacketPayload) error {
	frame, err := DecodeMISB0601Packet(payload)
	if err != nil {
		c.state.metrics.recordError(err)
		return err
	}
	frame.Source = c.cfg.Source
	data, err := marshalBaseMessage(MISB0601FrameType, frame, c.cfg.Name, frame.ReceivedAt)
	if err != nil {
		c.state.metrics.recordError(err)
		return err
	}
	if c.cfg.Bus == nil {
		err := errors.New("KLV decoder component requires a bus to publish decoded frames")
		c.state.metrics.recordError(err)
		return err
	}
	if err := c.cfg.Bus.Publish(ctx, c.cfg.FrameSubject, data); err != nil {
		c.state.metrics.recordError(err)
		return err
	}
	c.state.metrics.recordMessage(len(payload.PacketBytes), c.cfg.Clock().UTC())
	return nil
}

type ProjectorConfig struct {
	Name         string
	FrameSubject string
	Owner        string
	WriteTimeout time.Duration
	WriteRetries int
	Clock        func() time.Time
}

type ProjectorComponent struct {
	cfg   ProjectorConfig
	state componentState
}

func NewProjectorComponent(cfg ProjectorConfig) (*ProjectorComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-klv-project"
	}
	if cfg.FrameSubject == "" {
		cfg.FrameSubject = DefaultFrameSubject
	}
	if cfg.Owner == "" {
		cfg.Owner = DefaultOwner
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = DefaultWriteTimeout
	}
	if cfg.WriteTimeout < 0 {
		return nil, fmt.Errorf("KLV projector write_timeout must be greater than zero")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &ProjectorComponent{cfg: cfg, state: newComponentState(cfg.Clock)}, nil
}

func (c *ProjectorComponent) Initialize() error {
	return c.state.Initialize()
}

func (c *ProjectorComponent) Start(ctx context.Context) error {
	return c.state.Start(ctx)
}

func (c *ProjectorComponent) Stop(timeout time.Duration) error {
	return c.state.Stop(timeout)
}

func (c *ProjectorComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "KLV governed graph projection processor",
		Version:     "v0.1.0",
	}
}

func (c *ProjectorComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("misb0601_frames", component.DirectionInput, c.cfg.FrameSubject, MISB0601FrameType)}
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
				Subject: SubjectEntityCreateWithTriples,
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
			Description: "SemStreams graph update mutation request",
			Config: component.NATSRequestPort{
				Subject: SubjectEntityUpdateWithTriples,
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
			"frame_subject": stringProperty("SemStreams subject carrying decoded MISB ST 0601 frames", c.cfg.FrameSubject),
			"owner":         stringProperty("SemStreams projection owner bound through registry/heartbeat", c.cfg.Owner),
			"write_timeout": stringProperty("Graph mutation request timeout", c.outputTimeout().String()),
		},
		Required: []string{"frame_subject", "owner"},
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
	return DefaultWriteTimeout
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

func (s *componentState) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
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

func (s *componentState) Stop(_ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == component.StateStarted || s.state == component.StateInitialized {
		s.state = component.StateStopped
	}
	return nil
}

func (s *componentState) Current() component.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *componentState) Health() component.HealthStatus {
	s.mu.Lock()
	state := s.state
	clock := s.clock
	s.mu.Unlock()
	return s.metrics.healthAt(state, clock().UTC())
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

func (m *flowCounters) recordMessage(size int, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages++
	if size > 0 {
		m.bytes += uint64(size)
	}
	m.lastActivity = now.UTC()
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

func uptimeSinceAt(startedAt, now time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	if now.Before(startedAt) {
		return 0
	}
	return now.Sub(startedAt)
}
