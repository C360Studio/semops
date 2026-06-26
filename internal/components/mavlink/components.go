package mavlink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const (
	DefaultUDPListenAddr       = ":14550"
	DefaultUDPMaxDatagramBytes = 4096
	DefaultUDPReadInterval     = 250 * time.Millisecond
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
		return errors.New("mavlink NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("mavlink NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type UDPInputConfig struct {
	Name              string
	Source            string
	ListenAddr        string
	RawSubject        string
	MaxDatagramBytes  int
	ReadInterval      time.Duration
	Clock             func() time.Time
	OpenedListener    *net.UDPConn
	AdvertisedHost    string
	AdvertisedUDPPort int
}

type UDPInputComponent struct {
	cfg UDPInputConfig
	bus Bus

	mu      sync.Mutex
	state   component.State
	conn    *net.UDPConn
	cancel  context.CancelFunc
	done    chan error
	metrics flowCounters
}

func NewUDPInputComponent(cfg UDPInputConfig, bus Bus) (*UDPInputComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-input-mavlink-udp"
	}
	if cfg.Source == "" {
		cfg.Source = "mavlink:udp"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = DefaultUDPListenAddr
	}
	if cfg.RawSubject == "" {
		cfg.RawSubject = DefaultRawSubject
	}
	if cfg.MaxDatagramBytes == 0 {
		cfg.MaxDatagramBytes = DefaultUDPMaxDatagramBytes
	}
	if cfg.MaxDatagramBytes < 0 {
		return nil, fmt.Errorf("mavlink UDP max datagram bytes must be greater than zero")
	}
	if cfg.ReadInterval == 0 {
		cfg.ReadInterval = DefaultUDPReadInterval
	}
	if cfg.ReadInterval < 0 {
		return nil, fmt.Errorf("mavlink UDP read interval must be greater than zero")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("mavlink UDP input component requires a bus")
	}
	return &UDPInputComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *UDPInputComponent) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	return nil
}

func (c *UDPInputComponent) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	if c.state == component.StateStarted {
		c.mu.Unlock()
		return nil
	}
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	conn := c.cfg.OpenedListener
	if conn == nil {
		addr, err := net.ResolveUDPAddr("udp", c.cfg.ListenAddr)
		if err != nil {
			c.state = component.StateFailed
			c.mu.Unlock()
			return fmt.Errorf("resolve MAVLink UDP input address: %w", err)
		}
		conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			c.state = component.StateFailed
			c.mu.Unlock()
			return fmt.Errorf("listen MAVLink UDP input: %w", err)
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.conn = conn
	c.cancel = cancel
	c.done = make(chan error, 1)
	c.state = component.StateStarted
	c.metrics.startedAt = c.cfg.Clock().UTC()
	c.mu.Unlock()

	go func() {
		c.done <- c.run(runCtx)
	}()
	return nil
}

func (c *UDPInputComponent) Stop(timeout time.Duration) error {
	c.mu.Lock()
	if c.state != component.StateStarted {
		c.mu.Unlock()
		return nil
	}
	cancel := c.cancel
	conn := c.conn
	done := c.done
	c.state = component.StateStopped
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if conn != nil {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
	}
	if done == nil {
		return nil
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("stop MAVLink UDP input timed out after %s", timeout)
	}
}

func (c *UDPInputComponent) run(ctx context.Context) error {
	buffer := make([]byte, c.cfg.MaxDatagramBytes)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadInterval)); err != nil {
			c.recordError(err)
			return fmt.Errorf("set MAVLink UDP input read deadline: %w", err)
		}
		n, remote, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			c.recordError(err)
			return fmt.Errorf("read MAVLink UDP datagram: %w", err)
		}
		remoteAddr := ""
		if remote != nil {
			remoteAddr = remote.String()
		}
		if err := c.PublishFrame(ctx, buffer[:n], remoteAddr); err != nil {
			c.recordError(err)
		}
	}
}

func (c *UDPInputComponent) PublishFrame(ctx context.Context, frame []byte, remoteAddr string) error {
	if len(frame) == 0 {
		err := errors.New("MAVLink UDP input received empty frame")
		c.recordError(err)
		return err
	}
	now := c.cfg.Clock().UTC()
	payload := NewRawFramePayload(c.cfg.Source, remoteAddr, now, frame)
	data, err := marshalBaseMessage(RawFrameType, payload, c.cfg.Name, now)
	if err != nil {
		c.recordError(err)
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.RawSubject, data); err != nil {
		c.recordError(err)
		return err
	}
	c.recordMessage(len(frame), now)
	return nil
}

func (c *UDPInputComponent) Addr() net.Addr {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	return c.conn.LocalAddr()
}

func (c *UDPInputComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "MAVLink UDP transport input component",
		Version:     "v0.1.0",
	}
}

func (c *UDPInputComponent) InputPorts() []component.Port {
	host, port := splitUDPListen(c.cfg.ListenAddr)
	if c.cfg.AdvertisedHost != "" {
		host = c.cfg.AdvertisedHost
	}
	if c.cfg.AdvertisedUDPPort > 0 {
		port = c.cfg.AdvertisedUDPPort
	}
	return []component.Port{{
		Name:        "mavlink_datagrams",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Native MAVLink UDP datagram ingress",
		Config: component.NetworkPort{
			Protocol: "udp",
			Host:     host,
			Port:     port,
		},
	}}
}

func (c *UDPInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_frames", component.DirectionOutput, c.cfg.RawSubject, RawFrameType)}
}

func (c *UDPInputComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"listen_addr":        stringProperty("UDP address for native MAVLink datagram input", c.cfg.ListenAddr),
			"raw_subject":        stringProperty("SemStreams subject carrying raw MAVLink frames", c.cfg.RawSubject),
			"source":             stringProperty("Source label recorded in raw MAVLink payloads", c.cfg.Source),
			"max_datagram_bytes": intProperty("Maximum accepted UDP datagram size", c.cfg.MaxDatagramBytes),
		},
		Required: []string{"listen_addr", "raw_subject", "source"},
	}
}

func (c *UDPInputComponent) Health() component.HealthStatus {
	return c.metrics.health(c.state)
}

func (c *UDPInputComponent) DataFlow() component.FlowMetrics {
	return c.metrics.flow()
}

type DecoderConfig struct {
	Name           string
	Source         string
	RawSubject     string
	DecodedSubject string
	RawMaxRecords  int
	RawMaxBytes    int
	Clock          func() time.Time
	Registry       *payloadregistry.Registry
	Parser         *mavcodec.Parser
	RawLane        *mavcodec.RawLane
}

type DecoderComponent struct {
	cfg     DecoderConfig
	bus     Bus
	decoder *message.Decoder

	mu           sync.Mutex
	state        component.State
	subscription Subscription
	metrics      flowCounters
}

func NewDecoderComponent(cfg DecoderConfig, bus Bus) (*DecoderComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-mavlink-decode"
	}
	if cfg.Source == "" {
		cfg.Source = "mavlink:decoder"
	}
	if cfg.RawSubject == "" {
		cfg.RawSubject = DefaultRawSubject
	}
	if cfg.DecodedSubject == "" {
		cfg.DecodedSubject = DefaultDecodedSubject
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.Parser == nil {
		cfg.Parser = mavcodec.NewParser()
	}
	if cfg.RawLane == nil {
		cfg.RawLane = mavcodec.NewRawLane(mavcodec.RawLaneConfig{
			Source:     cfg.Source,
			MaxRecords: cfg.RawMaxRecords,
			MaxBytes:   cfg.RawMaxBytes,
			Clock:      cfg.Clock,
		})
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if bus == nil {
		return nil, fmt.Errorf("mavlink decoder component requires a bus")
	}
	return &DecoderComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *DecoderComponent) Initialize() error {
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

func (c *DecoderComponent) Start(ctx context.Context) error {
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

	sub, err := c.bus.Subscribe(ctx, c.cfg.RawSubject, func(msgCtx context.Context, msg *nats.Msg) {
		if err := c.HandleRawMessage(msgCtx, msg.Data); err != nil {
			c.recordError(err)
		}
	})
	if err != nil {
		c.markFailed(err)
		return fmt.Errorf("subscribe MAVLink decoder raw subject: %w", err)
	}
	c.mu.Lock()
	c.subscription = sub
	c.state = component.StateStarted
	c.metrics.startedAt = c.cfg.Clock().UTC()
	c.mu.Unlock()
	return nil
}

func (c *DecoderComponent) Stop(time.Duration) error {
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

func (c *DecoderComponent) HandleRawMessage(ctx context.Context, data []byte) error {
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	envelope, err := c.decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("decode MAVLink raw BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*RawFramePayload)
	if !ok {
		return fmt.Errorf("MAVLink decoder received payload %T, want *RawFramePayload", envelope.Payload())
	}
	return c.HandleRawPayload(ctx, payload)
}

func (c *DecoderComponent) HandleRawPayload(ctx context.Context, payload *RawFramePayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	packets, parseErr := c.cfg.Parser.Parse(payload.Frame)
	if parseErr != nil {
		record, captureErr := c.cfg.RawLane.Capture(payload.Frame, nil)
		if captureErr != nil {
			return fmt.Errorf("capture unparsable MAVLink frame: %w", captureErr)
		}
		return fmt.Errorf("parse MAVLink frame %s: %w", record.Ref, parseErr)
	}
	if len(packets) != 1 {
		record, captureErr := c.cfg.RawLane.Capture(payload.Frame, nil)
		if captureErr != nil {
			return fmt.Errorf("capture invalid MAVLink frame: %w", captureErr)
		}
		return fmt.Errorf("expected exactly one valid MAVLink packet for %s, got %d", record.Ref, len(packets))
	}
	packet := packets[0]
	record, err := c.cfg.RawLane.Capture(payload.Frame, packet)
	if err != nil {
		return fmt.Errorf("capture MAVLink frame: %w", err)
	}
	decoded := NewDecodedPacketPayload(c.cfg.Source, record)
	data, err := marshalBaseMessage(DecodedPacketType, decoded, c.cfg.Name, record.ReceivedAt)
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.DecodedSubject, data); err != nil {
		return err
	}
	c.recordMessage(len(payload.Frame), c.cfg.Clock().UTC())
	return nil
}

func (c *DecoderComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "MAVLink raw-frame decoder processor",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("raw_frames", component.DirectionInput, c.cfg.RawSubject, RawFrameType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("decoded_packets", component.DirectionOutput, c.cfg.DecodedSubject, DecodedPacketType)}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_subject":     stringProperty("SemStreams subject carrying raw MAVLink frames", c.cfg.RawSubject),
			"decoded_subject": stringProperty("SemStreams subject carrying decoded MAVLink packets", c.cfg.DecodedSubject),
			"source":          stringProperty("Source label recorded in decoded MAVLink payloads", c.cfg.Source),
			"raw_max_records": intProperty("Maximum retained raw frame records before eviction", rawLaneRecords(c.cfg.RawMaxRecords)),
			"raw_max_bytes":   intProperty("Maximum retained raw frame bytes before eviction", rawLaneBytes(c.cfg.RawMaxBytes)),
		},
		Required: []string{"raw_subject", "decoded_subject", "source"},
	}
}

func (c *DecoderComponent) Health() component.HealthStatus {
	return c.metrics.health(c.state)
}

func (c *DecoderComponent) DataFlow() component.FlowMetrics {
	return c.metrics.flow()
}

type ProjectorConfig struct {
	Name           string
	DecodedSubject string
	Registry       *payloadregistry.Registry
	Parser         *mavcodec.Parser
	Projector      *mavprojector.Projector
	Writer         PlanWriter
	WriteRetries   int
	WriteTimeout   time.Duration
	Clock          func() time.Time
}

type PlanWriter interface {
	Apply(ctx context.Context, plan mavprojector.Plan) error
}

type ProjectorComponent struct {
	cfg     ProjectorConfig
	bus     Bus
	decoder *message.Decoder

	mu           sync.Mutex
	projectMu    sync.Mutex
	state        component.State
	subscription Subscription
	metrics      flowCounters
}

func NewProjectorComponent(cfg ProjectorConfig, bus Bus) (*ProjectorComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-processor-mavlink-project"
	}
	if cfg.DecodedSubject == "" {
		cfg.DecodedSubject = DefaultDecodedSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Parser == nil {
		cfg.Parser = mavcodec.NewParser()
	}
	if cfg.Projector == nil {
		cfg.Projector = mavprojector.NewProjector(mavprojector.Config{})
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("mavlink projector component requires a plan writer")
	}
	if cfg.WriteRetries == 0 {
		cfg.WriteRetries = 4
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("mavlink projector component requires a bus")
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

	sub, err := c.bus.Subscribe(ctx, c.cfg.DecodedSubject, func(msgCtx context.Context, msg *nats.Msg) {
		if err := c.HandleDecodedMessage(msgCtx, msg.Data); err != nil {
			c.recordError(err)
		}
	})
	if err != nil {
		c.markFailed(err)
		return fmt.Errorf("subscribe MAVLink projector decoded subject: %w", err)
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

func (c *ProjectorComponent) HandleDecodedMessage(ctx context.Context, data []byte) error {
	if c.decoder == nil {
		if err := c.Initialize(); err != nil {
			return err
		}
	}
	envelope, err := c.decoder.Decode(data)
	if err != nil {
		return fmt.Errorf("decode MAVLink packet BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*DecodedPacketPayload)
	if !ok {
		return fmt.Errorf("MAVLink projector received payload %T, want *DecodedPacketPayload", envelope.Payload())
	}
	return c.HandleDecodedPayload(ctx, payload)
}

func (c *ProjectorComponent) HandleDecodedPayload(ctx context.Context, payload *DecodedPacketPayload) error {
	c.projectMu.Lock()
	defer c.projectMu.Unlock()

	packet, err := payload.Packet(c.cfg.Parser)
	if err != nil {
		return err
	}
	plan, err := c.cfg.Projector.ProjectPacket(packet)
	if err != nil {
		return fmt.Errorf("project MAVLink packet: %w", err)
	}
	if len(plan.Mutations) == 0 {
		c.recordMessage(len(payload.Frame), c.cfg.Clock().UTC())
		return nil
	}
	if err := c.writePlan(ctx, packet, plan); err != nil {
		return err
	}
	c.recordMessage(len(payload.Frame), c.cfg.Clock().UTC())
	return nil
}

func (c *ProjectorComponent) writePlan(ctx context.Context, packet *mavcodec.Packet, plan mavprojector.Plan) error {
	attempts := c.cfg.WriteRetries
	if attempts <= 0 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := c.cfg.Writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || entityID == "" || !c.cfg.Projector.MarkBornForPacket(packet, entityID) {
				return fmt.Errorf("write MAVLink graph plan: %w", err)
			}
			next, projectErr := c.cfg.Projector.ProjectPacket(packet)
			if projectErr != nil {
				return fmt.Errorf("reproject MAVLink packet after birth reconciliation: %w", projectErr)
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
	return fmt.Errorf("MAVLink graph birth reconciliation exceeded retry limit")
}

func (c *ProjectorComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "MAVLink governed graph projection processor",
		Version:     "v0.1.0",
	}
}

func (c *ProjectorComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("decoded_packets", component.DirectionInput, c.cfg.DecodedSubject, DecodedPacketType)}
}

func (c *ProjectorComponent) OutputPorts() []component.Port {
	timeout := c.cfg.WriteTimeout
	if timeout <= 0 {
		timeout = mavprojector.DefaultWriteTimeout
	}
	return []component.Port{
		{
			Name:        "graph_create",
			Direction:   component.DirectionOutput,
			Required:    true,
			Description: "SemStreams born-first graph mutation request",
			Config: component.NATSRequestPort{
				Subject: mavprojector.SubjectEntityCreateWithTriples,
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
				Subject: mavprojector.SubjectEntityUpdateWithTriples,
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
			"decoded_subject": stringProperty("SemStreams subject carrying decoded MAVLink packets", c.cfg.DecodedSubject),
			"owner":           stringProperty("SemStreams projection owner bound through registry/heartbeat", cop.OwnerMAVLink),
			"write_timeout":   stringProperty("Graph mutation request timeout", c.outputTimeout().String()),
		},
		Required: []string{"decoded_subject", "owner"},
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
	return mavprojector.DefaultWriteTimeout
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
	healthy := state == component.StateStarted || state == component.StateInitialized
	return component.HealthStatus{
		Healthy:    healthy && m.lastError == "",
		LastCheck:  time.Now().UTC(),
		ErrorCount: m.errors,
		LastError:  m.lastError,
		Uptime:     uptimeSince(m.startedAt),
		Status:     state.String(),
	}
}

func (m *flowCounters) flow() component.FlowMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	elapsed := time.Since(m.startedAt).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	return component.FlowMetrics{
		MessagesPerSecond: float64(m.messages) / elapsed,
		BytesPerSecond:    float64(m.bytes) / elapsed,
		ErrorRate:         float64(m.errors) / elapsed,
		LastActivity:      m.lastActivity,
	}
}

func uptimeSince(startedAt time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return time.Since(startedAt)
}

func (c *UDPInputComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *UDPInputComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *DecoderComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *DecoderComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *DecoderComponent) markFailed(err error) {
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

func (c *ProjectorComponent) markFailed(err error) {
	c.mu.Lock()
	c.state = component.StateFailed
	c.mu.Unlock()
	c.recordError(err)
}

func marshalBaseMessage(msgType message.Type, payload message.Payload, source string, observedAt time.Time) ([]byte, error) {
	envelope := message.NewBaseMessage(msgType, payload, source, message.WithTime(observedAt.UTC()))
	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal %s BaseMessage: %w", msgType.Key(), err)
	}
	return data, nil
}

func streamPort(name string, direction component.Direction, subject string, msgType message.Type) component.Port {
	return streamPortWithQueue(name, direction, subject, "", msgType)
}

func streamPortWithQueue(
	name string,
	direction component.Direction,
	subject string,
	queue string,
	msgType message.Type,
) component.Port {
	return component.Port{
		Name:        name,
		Direction:   direction,
		Required:    true,
		Description: fmt.Sprintf("%s %s", msgType.Key(), name),
		Config: component.NATSPort{
			Subject: subject,
			Queue:   queue,
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

func intProperty(description string, fallback int) component.PropertySchema {
	return component.PropertySchema{Type: "int", Description: description, Default: fallback}
}

func rawLaneRecords(value int) int {
	if value > 0 {
		return value
	}
	return mavcodec.DefaultRawLaneMaxRecords
}

func rawLaneBytes(value int) int {
	if value > 0 {
		return value
	}
	return mavcodec.DefaultRawLaneMaxBytes
}

func splitUDPListen(listenAddr string) (string, int) {
	host, portText, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "0.0.0.0", 0
	}
	port, err := net.LookupPort("udp", portText)
	if err != nil {
		return host, 0
	}
	if host == "" {
		host = "0.0.0.0"
	}
	return host, port
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
