package cot

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const (
	DefaultUDPListenAddr       = ":8087"
	DefaultUDPMaxDatagramBytes = 64 * 1024
	DefaultUDPReadInterval     = 250 * time.Millisecond
	DefaultTCPMaxEventBytes    = 1024 * 1024
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
		return errors.New("cot NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("cot NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type ReplayAppender interface {
	Append(record cotcodec.RawEventRecord) error
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
		cfg.Name = "semops-input-cot-udp"
	}
	if cfg.Source == "" {
		cfg.Source = "cot:udp"
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
		return nil, fmt.Errorf("cot UDP max datagram bytes must be greater than zero")
	}
	if cfg.ReadInterval == 0 {
		cfg.ReadInterval = DefaultUDPReadInterval
	}
	if cfg.ReadInterval < 0 {
		return nil, fmt.Errorf("cot UDP read interval must be greater than zero")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("cot UDP input component requires a bus")
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
			return fmt.Errorf("resolve CoT UDP input address: %w", err)
		}
		listener, err := net.ListenUDP("udp", addr)
		if err != nil {
			c.state = component.StateFailed
			c.mu.Unlock()
			return fmt.Errorf("listen CoT UDP input: %w", err)
		}
		conn = listener
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
	return waitDone(done, timeout, "stop CoT UDP input")
}

func (c *UDPInputComponent) run(ctx context.Context) error {
	buffer := make([]byte, c.cfg.MaxDatagramBytes)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := c.conn.SetReadDeadline(time.Now().Add(c.cfg.ReadInterval)); err != nil {
			c.recordError(err)
			return fmt.Errorf("set CoT UDP input read deadline: %w", err)
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
			return fmt.Errorf("read CoT UDP datagram: %w", err)
		}
		remoteAddr := ""
		if remote != nil {
			remoteAddr = remote.String()
		}
		if err := c.PublishEvent(ctx, buffer[:n], remoteAddr); err != nil {
			c.recordError(err)
		}
	}
}

func (c *UDPInputComponent) PublishEvent(ctx context.Context, rawXML []byte, remoteAddr string) error {
	return publishRawEvent(ctx, c.bus, c.cfg.RawSubject, c.cfg.Name, c.cfg.Source, remoteAddr, c.cfg.Clock, rawXML, &c.metrics)
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
		Description: "CoT UDP transport input component",
		Version:     "v0.1.0",
	}
}

func (c *UDPInputComponent) InputPorts() []component.Port {
	host, port := splitListen("udp", c.cfg.ListenAddr)
	if c.cfg.AdvertisedHost != "" {
		host = c.cfg.AdvertisedHost
	}
	if c.cfg.AdvertisedUDPPort > 0 {
		port = c.cfg.AdvertisedUDPPort
	}
	return []component.Port{{
		Name:        "cot_datagrams",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Native CoT UDP datagram ingress",
		Config: component.NetworkPort{
			Protocol: "udp",
			Host:     host,
			Port:     port,
		},
	}}
}

func (c *UDPInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_events", component.DirectionOutput, c.cfg.RawSubject, RawEventType)}
}

func (c *UDPInputComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"listen_addr":         stringProperty("UDP address for native CoT datagram input", c.cfg.ListenAddr),
			"raw_subject":         stringProperty("SemStreams subject carrying raw CoT events", c.cfg.RawSubject),
			"source":              stringProperty("Source label recorded in raw CoT payloads", c.cfg.Source),
			"max_datagram_bytes":  intProperty("Maximum accepted UDP datagram size", c.cfg.MaxDatagramBytes),
			"advertised_udp_port": intProperty("Advertised UDP ingress port for flowgraph metadata", c.cfg.AdvertisedUDPPort),
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

type TCPInputConfig struct {
	Name              string
	Source            string
	ListenAddr        string
	RawSubject        string
	MaxEventBytes     int
	Clock             func() time.Time
	OpenedListener    net.Listener
	AdvertisedHost    string
	AdvertisedTCPPort int
}

type TCPInputComponent struct {
	cfg TCPInputConfig
	bus Bus

	mu       sync.Mutex
	state    component.State
	listener net.Listener
	cancel   context.CancelFunc
	done     chan error
	metrics  flowCounters
}

func NewTCPInputComponent(cfg TCPInputConfig, bus Bus) (*TCPInputComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-input-cot-tcp"
	}
	if cfg.Source == "" {
		cfg.Source = "cot:tcp"
	}
	if cfg.RawSubject == "" {
		cfg.RawSubject = DefaultRawSubject
	}
	if cfg.MaxEventBytes == 0 {
		cfg.MaxEventBytes = DefaultTCPMaxEventBytes
	}
	if cfg.MaxEventBytes < 0 {
		return nil, fmt.Errorf("cot TCP max event bytes must be greater than zero")
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("cot TCP input component requires a bus")
	}
	return &TCPInputComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *TCPInputComponent) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	return nil
}

func (c *TCPInputComponent) Start(ctx context.Context) error {
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
	listener := c.cfg.OpenedListener
	if listener == nil {
		if c.cfg.ListenAddr == "" {
			c.state = component.StateFailed
			c.mu.Unlock()
			return fmt.Errorf("cot TCP input component requires a listen address")
		}
		var err error
		listener, err = net.Listen("tcp", c.cfg.ListenAddr)
		if err != nil {
			c.state = component.StateFailed
			c.mu.Unlock()
			return fmt.Errorf("listen CoT TCP input: %w", err)
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.listener = listener
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

func (c *TCPInputComponent) Stop(timeout time.Duration) error {
	c.mu.Lock()
	if c.state != component.StateStarted {
		c.mu.Unlock()
		return nil
	}
	cancel := c.cancel
	listener := c.listener
	done := c.done
	c.state = component.StateStopped
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return err
		}
	}
	return waitDone(done, timeout, "stop CoT TCP input")
}

func (c *TCPInputComponent) run(ctx context.Context) error {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			c.recordError(err)
			return fmt.Errorf("accept CoT TCP connection: %w", err)
		}
		go c.handleConn(ctx, conn)
	}
}

func (c *TCPInputComponent) handleConn(ctx context.Context, conn net.Conn) {
	done := make(chan struct{})
	defer close(done)
	defer conn.Close()
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), c.cfg.MaxEventBytes)
	for scanner.Scan() {
		if err := c.PublishEvent(ctx, scanner.Bytes(), conn.RemoteAddr().String()); err != nil {
			c.recordError(err)
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		c.recordError(err)
	}
}

func (c *TCPInputComponent) PublishEvent(ctx context.Context, rawXML []byte, remoteAddr string) error {
	return publishRawEvent(ctx, c.bus, c.cfg.RawSubject, c.cfg.Name, c.cfg.Source, remoteAddr, c.cfg.Clock, rawXML, &c.metrics)
}

func (c *TCPInputComponent) Addr() net.Addr {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.listener == nil {
		return nil
	}
	return c.listener.Addr()
}

func (c *TCPInputComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "CoT TCP transport input component",
		Version:     "v0.1.0",
	}
}

func (c *TCPInputComponent) InputPorts() []component.Port {
	host, port := splitListen("tcp", c.cfg.ListenAddr)
	if c.cfg.AdvertisedHost != "" {
		host = c.cfg.AdvertisedHost
	}
	if c.cfg.AdvertisedTCPPort > 0 {
		port = c.cfg.AdvertisedTCPPort
	}
	return []component.Port{{
		Name:        "cot_stream",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Native newline-delimited CoT TCP ingress",
		Config: component.NetworkPort{
			Protocol: "tcp",
			Host:     host,
			Port:     port,
		},
	}}
}

func (c *TCPInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_events", component.DirectionOutput, c.cfg.RawSubject, RawEventType)}
}

func (c *TCPInputComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"listen_addr":         stringProperty("TCP address for native CoT event input", c.cfg.ListenAddr),
			"raw_subject":         stringProperty("SemStreams subject carrying raw CoT events", c.cfg.RawSubject),
			"source":              stringProperty("Source label recorded in raw CoT payloads", c.cfg.Source),
			"max_event_bytes":     intProperty("Maximum accepted newline-delimited CoT event size", c.cfg.MaxEventBytes),
			"advertised_tcp_port": intProperty("Advertised TCP ingress port for flowgraph metadata", c.cfg.AdvertisedTCPPort),
		},
		Required: []string{"listen_addr", "raw_subject", "source"},
	}
}

func (c *TCPInputComponent) Health() component.HealthStatus {
	return c.metrics.health(c.state)
}

func (c *TCPInputComponent) DataFlow() component.FlowMetrics {
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
	RawLane        *cotcodec.RawLane
	Replay         ReplayAppender
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
		cfg.Name = "semops-processor-cot-decode"
	}
	if cfg.Source == "" {
		cfg.Source = "cot:decoder"
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
	if cfg.RawLane == nil {
		cfg.RawLane = cotcodec.NewRawLane(cotcodec.RawLaneConfig{
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
		return nil, fmt.Errorf("cot decoder component requires a bus")
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
		return fmt.Errorf("subscribe CoT decoder raw subject: %w", err)
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
		return fmt.Errorf("decode CoT raw BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*RawEventPayload)
	if !ok {
		return fmt.Errorf("CoT decoder received payload %T, want *RawEventPayload", envelope.Payload())
	}
	return c.HandleRawPayload(ctx, payload)
}

func (c *DecoderComponent) HandleRawPayload(ctx context.Context, payload *RawEventPayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	event, parseErr := cotcodec.Unmarshal(payload.RawXML)
	if parseErr != nil {
		record, captureErr := c.cfg.RawLane.Capture(payload.RawXML, nil)
		if captureErr != nil {
			return fmt.Errorf("capture unparsable CoT event: %w", captureErr)
		}
		if err := c.appendReplay(record); err != nil {
			return err
		}
		return fmt.Errorf("parse CoT event %s: %w", record.Ref, parseErr)
	}
	record, err := c.cfg.RawLane.Capture(payload.RawXML, &event)
	if err != nil {
		return fmt.Errorf("capture CoT event: %w", err)
	}
	if err := c.appendReplay(record); err != nil {
		return err
	}
	decoded := NewDecodedEventPayload(c.cfg.Source, record, event)
	data, err := marshalBaseMessage(DecodedEventType, decoded, c.cfg.Name, record.ReceivedAt)
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.DecodedSubject, data); err != nil {
		return err
	}
	c.recordMessage(len(payload.RawXML), c.cfg.Clock().UTC())
	return nil
}

func (c *DecoderComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "CoT raw-event decoder processor",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("raw_events", component.DirectionInput, c.cfg.RawSubject, RawEventType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("decoded_events", component.DirectionOutput, c.cfg.DecodedSubject, DecodedEventType)}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_subject":     stringProperty("SemStreams subject carrying raw CoT events", c.cfg.RawSubject),
			"decoded_subject": stringProperty("SemStreams subject carrying decoded CoT events", c.cfg.DecodedSubject),
			"source":          stringProperty("Source label recorded in decoded CoT payloads", c.cfg.Source),
			"raw_max_records": intProperty("Maximum retained raw CoT records before eviction", rawLaneRecords(c.cfg.RawMaxRecords)),
			"raw_max_bytes":   intProperty("Maximum retained raw CoT bytes before eviction", rawLaneBytes(c.cfg.RawMaxBytes)),
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

func (c *DecoderComponent) appendReplay(record cotcodec.RawEventRecord) error {
	if c.cfg.Replay == nil {
		return nil
	}
	if err := c.cfg.Replay.Append(record); err != nil {
		return fmt.Errorf("append CoT replay record %q: %w", record.Ref, err)
	}
	return nil
}

type ProjectorConfig struct {
	Name           string
	DecodedSubject string
	Registry       *payloadregistry.Registry
	Projector      *cotprojector.Projector
	Writer         PlanWriter
	WriteRetries   int
	WriteTimeout   time.Duration
	Clock          func() time.Time
}

type PlanWriter interface {
	Apply(ctx context.Context, plan cotprojector.Plan) error
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
		cfg.Name = "semops-processor-cot-project"
	}
	if cfg.DecodedSubject == "" {
		cfg.DecodedSubject = DefaultDecodedSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Projector == nil {
		cfg.Projector = cotprojector.NewProjector(cotprojector.Config{})
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("cot projector component requires a plan writer")
	}
	if cfg.WriteRetries == 0 {
		cfg.WriteRetries = 4
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("cot projector component requires a bus")
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
		return fmt.Errorf("subscribe CoT projector decoded subject: %w", err)
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
		return fmt.Errorf("decode CoT event BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*DecodedEventPayload)
	if !ok {
		return fmt.Errorf("CoT projector received payload %T, want *DecodedEventPayload", envelope.Payload())
	}
	return c.HandleDecodedPayload(ctx, payload)
}

func (c *ProjectorComponent) HandleDecodedPayload(ctx context.Context, payload *DecodedEventPayload) error {
	event, err := payload.CoTEvent()
	if err != nil {
		return err
	}
	plan, err := c.cfg.Projector.ProjectEvent(event, payload.RawRef)
	if err != nil {
		return fmt.Errorf("project CoT event: %w", err)
	}
	if len(plan.Mutations) == 0 {
		c.recordMessage(len(payload.RawRef), c.cfg.Clock().UTC())
		return nil
	}
	if err := c.writePlan(ctx, event, payload.RawRef, plan); err != nil {
		return err
	}
	c.recordMessage(len(payload.RawRef), c.cfg.Clock().UTC())
	return nil
}

func (c *ProjectorComponent) writePlan(
	ctx context.Context,
	event cotcodec.Event,
	sourceRef string,
	plan cotprojector.Plan,
) error {
	attempts := c.cfg.WriteRetries
	if attempts <= 0 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := c.cfg.Writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || !c.cfg.Projector.MarkBornForEvent(event, entityID) {
				return fmt.Errorf("write CoT graph plan: %w", err)
			}
			next, projectErr := c.cfg.Projector.ProjectEvent(event, sourceRef)
			if projectErr != nil {
				return fmt.Errorf("reproject CoT event after birth reconciliation: %w", projectErr)
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
	return fmt.Errorf("CoT graph birth reconciliation exceeded retry limit")
}

func (c *ProjectorComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "CoT governed graph projection processor",
		Version:     "v0.1.0",
	}
}

func (c *ProjectorComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("decoded_events", component.DirectionInput, c.cfg.DecodedSubject, DecodedEventType)}
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
				Subject: cotprojector.SubjectEntityCreateWithTriples,
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
				Subject: cotprojector.SubjectEntityUpdateWithTriples,
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
			"decoded_subject": stringProperty("SemStreams subject carrying decoded CoT events", c.cfg.DecodedSubject),
			"owner":           stringProperty("SemStreams projection owner bound through registry/heartbeat", cop.OwnerTAK),
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
	return cotprojector.DefaultWriteTimeout
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

func (m *flowCounters) recordRawPublish(size int, now time.Time) {
	m.recordMessage(size, now)
}

func (m *flowCounters) recordRawError(err error) {
	m.recordError(err)
}

func uptimeSince(startedAt time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return time.Since(startedAt)
}

func (c *UDPInputComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *TCPInputComponent) recordError(err error) {
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

func publishRawEvent(
	ctx context.Context,
	bus Bus,
	subject string,
	componentName string,
	source string,
	remoteAddr string,
	clock func() time.Time,
	rawXML []byte,
	metrics *flowCounters,
) error {
	if len(rawXML) == 0 {
		err := errors.New("CoT input received empty event")
		metrics.recordRawError(err)
		return err
	}
	now := clock().UTC()
	payload := NewRawEventPayload(source, remoteAddr, now, rawXML)
	data, err := marshalBaseMessage(RawEventType, payload, componentName, now)
	if err != nil {
		metrics.recordRawError(err)
		return err
	}
	if err := bus.Publish(ctx, subject, data); err != nil {
		metrics.recordRawError(err)
		return err
	}
	metrics.recordRawPublish(len(rawXML), now)
	return nil
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

func intProperty(description string, fallback int) component.PropertySchema {
	return component.PropertySchema{Type: "int", Description: description, Default: fallback}
}

func rawLaneRecords(value int) int {
	if value > 0 {
		return value
	}
	return cotcodec.DefaultRawLaneMaxRecords
}

func rawLaneBytes(value int) int {
	if value > 0 {
		return value
	}
	return cotcodec.DefaultRawLaneMaxBytes
}

func splitListen(network, listenAddr string) (string, int) {
	host, portText, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "0.0.0.0", 0
	}
	port, err := net.LookupPort(network, portText)
	if err != nil {
		return host, 0
	}
	if host == "" {
		host = "0.0.0.0"
	}
	return host, port
}

func waitDone(done <-chan error, timeout time.Duration, action string) error {
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
		return fmt.Errorf("%s timed out after %s", action, timeout)
	}
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
