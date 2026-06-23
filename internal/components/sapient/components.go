package sapient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	sapientprojector "github.com/c360studio/semops/internal/projectors/sapient"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const (
	DefaultHTTPPollInterval     = 30 * time.Second
	DefaultHTTPMaxResponseBytes = 4 * 1024 * 1024
	DefaultHTTPStaleMultiplier  = 4
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
		return errors.New("sapient NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("sapient NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type ReplayAppender interface {
	Append(record sapientcodec.RawMessageRecord) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPInputConfig struct {
	Name             string
	Source           string
	URL              string
	Method           string
	RawSubject       string
	PollInterval     time.Duration
	StaleAfter       time.Duration
	ContactPolicy    string
	AuthRef          string
	MaxResponseBytes int
	Encoding         sapientcodec.Encoding
	Client           HTTPClient
	Clock            func() time.Time
}

type HTTPInputComponent struct {
	cfg HTTPInputConfig
	bus Bus

	mu      sync.Mutex
	state   component.State
	cancel  context.CancelFunc
	done    chan error
	metrics flowCounters

	lastProviderContact time.Time
	lastFreshData       time.Time
	lastStatusCode      int
	lastContentType     string
}

type HTTPInputDebugStatus struct {
	Endpoint            string        `json:"endpoint"`
	PollInterval        time.Duration `json:"poll_interval"`
	StaleAfter          time.Duration `json:"stale_after"`
	LastProviderContact time.Time     `json:"last_provider_contact,omitempty"`
	LastFreshData       time.Time     `json:"last_fresh_data,omitempty"`
	LastStatusCode      int           `json:"last_status_code,omitempty"`
	LastContentType     string        `json:"last_content_type,omitempty"`
}

func NewHTTPInputComponent(cfg HTTPInputConfig, bus Bus) (*HTTPInputComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-input-sapient-http"
	}
	if cfg.Source == "" {
		cfg.Source = "sapient:http"
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("sapient HTTP input component requires a URL")
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	cfg.Method = strings.ToUpper(cfg.Method)
	if cfg.RawSubject == "" {
		cfg.RawSubject = DefaultRawSubject
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = DefaultHTTPPollInterval
	}
	if cfg.PollInterval < 0 {
		return nil, fmt.Errorf("sapient HTTP poll interval must be greater than zero")
	}
	if cfg.StaleAfter == 0 {
		cfg.StaleAfter = time.Duration(DefaultHTTPStaleMultiplier) * cfg.PollInterval
	}
	if cfg.StaleAfter < 0 {
		return nil, fmt.Errorf("sapient HTTP stale_after must be greater than zero")
	}
	if cfg.MaxResponseBytes == 0 {
		cfg.MaxResponseBytes = DefaultHTTPMaxResponseBytes
	}
	if cfg.MaxResponseBytes < 0 {
		return nil, fmt.Errorf("sapient HTTP max response bytes must be greater than zero")
	}
	if cfg.Encoding != "" && !cfg.Encoding.Valid() {
		return nil, fmt.Errorf("sapient HTTP encoding %q is unsupported", cfg.Encoding)
	}
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("sapient HTTP input component requires a bus")
	}
	return &HTTPInputComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *HTTPInputComponent) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	return nil
}

func (c *HTTPInputComponent) Start(ctx context.Context) error {
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
	runCtx, cancel := context.WithCancel(ctx)
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

func (c *HTTPInputComponent) Stop(timeout time.Duration) error {
	c.mu.Lock()
	if c.state != component.StateStarted {
		c.mu.Unlock()
		return nil
	}
	cancel := c.cancel
	done := c.done
	c.state = component.StateStopped
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return waitDone(done, timeout, "stop SAPIENT HTTP input")
}

func (c *HTTPInputComponent) run(ctx context.Context) error {
	ticker := time.NewTicker(c.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.PollOnce(ctx); err != nil {
				c.recordError(err)
			}
		}
	}
}

func (c *HTTPInputComponent) PollOnce(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, c.cfg.Method, c.cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("build SAPIENT HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json, application/x-protobuf, application/octet-stream;q=0.8")
	if c.cfg.ContactPolicy != "" {
		req.Header.Set("User-Agent", c.cfg.ContactPolicy)
	}

	resp, err := c.cfg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("poll SAPIENT HTTP feed: %w", err)
	}
	defer resp.Body.Close()

	now := c.cfg.Clock().UTC()
	contentType := resp.Header.Get("Content-Type")
	c.recordProviderContact(resp.StatusCode, contentType, now)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("poll SAPIENT HTTP feed returned status %d", resp.StatusCode)
	}
	encoding, err := inferEncoding(c.cfg.Encoding, contentType)
	if err != nil {
		return err
	}
	body, err := readLimited(resp.Body, c.cfg.MaxResponseBytes)
	if err != nil {
		return err
	}
	payload := NewRawMessagePayload(c.cfg.Source, c.cfg.URL, now, resp.StatusCode, encoding, body)
	payload.ContentType = contentType
	if err := payload.Validate(); err != nil {
		return err
	}
	data, err := marshalBaseMessage(RawMessageType, payload, c.cfg.Name, now)
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.RawSubject, data); err != nil {
		return err
	}
	c.recordMessage(len(body), now)
	return nil
}

func (c *HTTPInputComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "SAPIENT HTTP preflight input component",
		Version:     "v0.1.0",
	}
}

func (c *HTTPInputComponent) InputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "sapient_feed",
			Direction:   component.DirectionInput,
			Required:    true,
			Description: "Outbound SAPIENT HTTP preflight dependency",
			Config: component.HTTPClientPort{
				Method:        c.cfg.Method,
				URLPattern:    c.cfg.URL,
				TriggerPort:   "poll_tick",
				AuthRef:       c.cfg.AuthRef,
				ContactPolicy: c.cfg.ContactPolicy,
				Interface: &component.InterfaceContract{
					Type:       "message.BaseMessage",
					Version:    "v1",
					Compatible: []string{RawMessageType.Key()},
				},
			},
		},
		{
			Name:        "poll_tick",
			Direction:   component.DirectionInput,
			Required:    true,
			Description: "Timer cadence for SAPIENT HTTP polling",
			Config: component.TimerPort{
				Interval: c.cfg.PollInterval.String(),
				Interface: &component.InterfaceContract{
					Type:    "timer.tick",
					Version: "v1",
				},
			},
		},
	}
}

func (c *HTTPInputComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_messages", component.DirectionOutput, c.cfg.RawSubject, RawMessageType)}
}

func (c *HTTPInputComponent) ConfigSchema() component.ConfigSchema {
	encodingDefault := string(c.cfg.Encoding)
	if encodingDefault == "" {
		encodingDefault = "auto"
	}
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"url":                stringProperty("SAPIENT or Apex HTTP preflight endpoint URL", c.cfg.URL),
			"method":             stringProperty("HTTP method for SAPIENT polling", c.cfg.Method),
			"raw_subject":        stringProperty("SemStreams subject carrying raw SAPIENT payloads", c.cfg.RawSubject),
			"source":             stringProperty("Source label recorded in raw SAPIENT payloads", c.cfg.Source),
			"encoding":           stringProperty("SAPIENT payload encoding: json, protobuf, or auto", encodingDefault),
			"poll_interval":      stringProperty("Timer cadence for SAPIENT HTTP polling", c.cfg.PollInterval.String()),
			"stale_after":        stringProperty("Maximum age of the last fresh SAPIENT payload", c.cfg.StaleAfter.String()),
			"contact_policy":     stringProperty("Public User-Agent/contact identity for feed providers", c.cfg.ContactPolicy),
			"auth_ref":           stringProperty("Secret reference for authenticated SAPIENT feeds", c.cfg.AuthRef),
			"max_response_bytes": intProperty("Maximum accepted SAPIENT HTTP response size", c.cfg.MaxResponseBytes),
		},
		Required: []string{"url", "method", "raw_subject", "source", "poll_interval"},
	}
}

func (c *HTTPInputComponent) Health() component.HealthStatus {
	c.mu.Lock()
	state := c.state
	staleAfter := c.cfg.StaleAfter
	clock := c.cfg.Clock
	lastFreshData := c.lastFreshData
	c.mu.Unlock()

	now := clock().UTC()
	health := c.metrics.healthAt(state, now)
	if staleAfter <= 0 || !health.Healthy {
		return health
	}
	if lastFreshData.IsZero() {
		lastFreshData = c.metrics.startedTime()
	}
	if lastFreshData.IsZero() {
		return health
	}
	age := now.Sub(lastFreshData)
	if age <= staleAfter {
		return health
	}
	health.Healthy = false
	health.Status = "stale"
	health.LastError = fmt.Sprintf("SAPIENT HTTP feed stale: no fresh payload for %s", age.Round(time.Second))
	return health
}

func (c *HTTPInputComponent) DataFlow() component.FlowMetrics {
	return c.metrics.flow()
}

func (c *HTTPInputComponent) DebugStatus() any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return HTTPInputDebugStatus{
		Endpoint:            c.cfg.URL,
		PollInterval:        c.cfg.PollInterval,
		StaleAfter:          c.cfg.StaleAfter,
		LastProviderContact: c.lastProviderContact,
		LastFreshData:       c.lastFreshData,
		LastStatusCode:      c.lastStatusCode,
		LastContentType:     c.lastContentType,
	}
}

type DecoderConfig struct {
	Name           string
	Source         string
	RawSubject     string
	DecodedSubject string
	Registry       *payloadregistry.Registry
	RawLane        *sapientcodec.RawLane
	RawMaxRecords  int
	RawMaxBytes    int
	Replay         ReplayAppender
	Descriptors    *sapientcodec.ProtoDescriptorSet
	Clock          func() time.Time
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
		cfg.Name = "semops-processor-sapient-decode"
	}
	if cfg.Source == "" {
		cfg.Source = "sapient"
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
	if cfg.RawLane == nil {
		cfg.RawLane = sapientcodec.NewRawLane(sapientcodec.RawLaneConfig{
			Source:     cfg.Source,
			MaxRecords: cfg.RawMaxRecords,
			MaxBytes:   cfg.RawMaxBytes,
			Clock:      cfg.Clock,
		})
	}
	if bus == nil {
		return nil, fmt.Errorf("sapient decoder component requires a bus")
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
		return fmt.Errorf("subscribe SAPIENT decoder raw subject: %w", err)
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
		return fmt.Errorf("decode SAPIENT raw BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*RawMessagePayload)
	if !ok {
		return fmt.Errorf("SAPIENT decoder received payload %T, want *RawMessagePayload", envelope.Payload())
	}
	return c.HandleRawPayload(ctx, payload)
}

func (c *DecoderComponent) HandleRawPayload(ctx context.Context, payload *RawMessagePayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	msg, parseErr := parsePayload(payload.Encoding, payload.RawPayload, c.cfg.Descriptors)
	if parseErr != nil {
		record, captureErr := c.cfg.RawLane.Capture(payload.RawPayload, payload.Encoding, nil)
		if captureErr != nil {
			return fmt.Errorf("capture unparsable SAPIENT payload: %w", captureErr)
		}
		if err := c.appendReplay(record); err != nil {
			return err
		}
		return fmt.Errorf("parse SAPIENT payload: %w", parseErr)
	}

	record, err := c.cfg.RawLane.Capture(payload.RawPayload, payload.Encoding, &msg)
	if err != nil {
		return fmt.Errorf("capture SAPIENT payload: %w", err)
	}
	if err := c.appendReplay(record); err != nil {
		return err
	}
	decoded := NewDecodedMessagePayload(record, msg)
	data, err := marshalBaseMessage(DecodedMessageType, decoded, c.cfg.Name, record.ReceivedAt)
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.DecodedSubject, data); err != nil {
		return err
	}
	c.recordMessage(len(payload.RawPayload), c.cfg.Clock().UTC())
	return nil
}

func (c *DecoderComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "SAPIENT raw-message preflight decoder processor",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("raw_messages", component.DirectionInput, c.cfg.RawSubject, RawMessageType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{
		streamPort("decoded_messages", component.DirectionOutput, c.cfg.DecodedSubject, DecodedMessageType),
	}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_subject":     stringProperty("SemStreams subject carrying raw SAPIENT payloads", c.cfg.RawSubject),
			"decoded_subject": stringProperty("SemStreams subject carrying decoded SAPIENT messages", c.cfg.DecodedSubject),
			"source":          stringProperty("Source token used for SAPIENT raw-lane refs", c.cfg.Source),
			"raw_max_records": intProperty(
				"Maximum SAPIENT raw-lane records retained in memory",
				rawLaneRecords(c.cfg.RawMaxRecords),
			),
			"raw_max_bytes": intProperty(
				"Maximum SAPIENT raw payload bytes retained in memory",
				rawLaneBytes(c.cfg.RawMaxBytes),
			),
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

func (c *DecoderComponent) appendReplay(record sapientcodec.RawMessageRecord) error {
	if c.cfg.Replay == nil {
		return nil
	}
	if err := c.cfg.Replay.Append(record); err != nil {
		return fmt.Errorf("append SAPIENT replay record %q: %w", record.Ref, err)
	}
	return nil
}

type ProjectorConfig struct {
	Name           string
	DecodedSubject string
	Registry       *payloadregistry.Registry
	Projector      *sapientprojector.Projector
	Writer         PlanWriter
	WriteRetries   int
	WriteTimeout   time.Duration
	Clock          func() time.Time
}

type PlanWriter interface {
	Apply(ctx context.Context, plan sapientprojector.Plan) error
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
		cfg.Name = "semops-processor-sapient-project"
	}
	if cfg.DecodedSubject == "" {
		cfg.DecodedSubject = DefaultDecodedSubject
	}
	if cfg.Registry == nil {
		cfg.Registry = payloadregistry.New()
	}
	if cfg.Projector == nil {
		cfg.Projector = sapientprojector.NewProjector(sapientprojector.Config{})
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("sapient projector component requires a plan writer")
	}
	if cfg.WriteRetries == 0 {
		cfg.WriteRetries = 4
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("sapient projector component requires a bus")
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
		return fmt.Errorf("subscribe SAPIENT projector decoded subject: %w", err)
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
		return fmt.Errorf("decode SAPIENT decoded BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*DecodedMessagePayload)
	if !ok {
		return fmt.Errorf("SAPIENT projector received payload %T, want *DecodedMessagePayload", envelope.Payload())
	}
	return c.HandleDecodedPayload(ctx, payload)
}

func (c *ProjectorComponent) HandleDecodedPayload(ctx context.Context, payload *DecodedMessagePayload) error {
	msg, err := payload.MessageCopy()
	if err != nil {
		return err
	}
	plan, err := c.cfg.Projector.ProjectMessage(msg, payload.RawRef)
	if err != nil {
		return fmt.Errorf("project SAPIENT message: %w", err)
	}
	if len(plan.Mutations) == 0 {
		c.recordMessage(len(payload.RawRef), c.cfg.Clock().UTC())
		return nil
	}
	if err := c.writePlan(ctx, msg, payload.RawRef, plan); err != nil {
		return err
	}
	c.recordMessage(len(payload.RawRef), c.cfg.Clock().UTC())
	return nil
}

func (c *ProjectorComponent) writePlan(
	ctx context.Context,
	msg sapientcodec.Message,
	sourceRef string,
	plan sapientprojector.Plan,
) error {
	attempts := c.cfg.WriteRetries
	if attempts <= 0 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if err := c.cfg.Writer.Apply(ctx, plan); err != nil {
			entityID, ok := entityAlreadyExists(err)
			if !ok || !c.cfg.Projector.MarkBornForMessage(msg, entityID) {
				return fmt.Errorf("write SAPIENT graph plan: %w", err)
			}
			next, projectErr := c.cfg.Projector.ProjectMessage(msg, sourceRef)
			if projectErr != nil {
				return fmt.Errorf("reproject SAPIENT message after birth reconciliation: %w", projectErr)
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
	return fmt.Errorf("SAPIENT graph birth reconciliation exceeded retry limit")
}

func (c *ProjectorComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "processor",
		Description: "SAPIENT governed graph projection processor",
		Version:     "v0.1.0",
	}
}

func (c *ProjectorComponent) InputPorts() []component.Port {
	return []component.Port{
		streamPort("decoded_messages", component.DirectionInput, c.cfg.DecodedSubject, DecodedMessageType),
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
				Subject: sapientprojector.SubjectEntityCreateWithTriples,
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
				Subject: sapientprojector.SubjectEntityUpdateWithTriples,
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
			"decoded_subject": stringProperty("SemStreams subject carrying decoded SAPIENT messages", c.cfg.DecodedSubject),
			"owner":           stringProperty("SemStreams projection owner bound through registry/heartbeat", cop.OwnerSAPIENT),
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
	return sapientprojector.DefaultWriteTimeout
}

func (c *DecoderComponent) markFailed(err error) {
	c.mu.Lock()
	c.state = component.StateFailed
	c.mu.Unlock()
	c.recordError(err)
}

func (c *ProjectorComponent) markFailed(err error) {
	c.mu.Lock()
	c.state = component.StateFailed
	c.mu.Unlock()
	c.recordError(err)
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

func (m *flowCounters) startedTime() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startedAt
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

func (c *HTTPInputComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
	c.mu.Lock()
	c.lastFreshData = now.UTC()
	c.mu.Unlock()
}

func (c *HTTPInputComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *HTTPInputComponent) recordProviderContact(statusCode int, contentType string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastProviderContact = now.UTC()
	c.lastStatusCode = statusCode
	c.lastContentType = contentType
}

func (c *DecoderComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *DecoderComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *ProjectorComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *ProjectorComponent) recordError(err error) {
	c.metrics.recordError(err)
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

func intProperty(description string, fallback int) component.PropertySchema {
	return component.PropertySchema{Type: "int", Description: description, Default: fallback}
}

func rawLaneRecords(value int) int {
	if value > 0 {
		return value
	}
	return sapientcodec.DefaultRawLaneMaxRecords
}

func rawLaneBytes(value int) int {
	if value > 0 {
		return value
	}
	return sapientcodec.DefaultRawLaneMaxBytes
}

func parsePayload(
	encoding sapientcodec.Encoding,
	data []byte,
	descriptors *sapientcodec.ProtoDescriptorSet,
) (sapientcodec.Message, error) {
	switch encoding {
	case sapientcodec.EncodingJSON:
		return sapientcodec.ParseJSONMessage(data)
	case sapientcodec.EncodingProtobuf:
		return sapientcodec.ParseBinaryMessage(data, descriptors)
	default:
		return sapientcodec.Message{}, fmt.Errorf("unsupported SAPIENT encoding %q", encoding)
	}
}

func inferEncoding(configured sapientcodec.Encoding, contentType string) (sapientcodec.Encoding, error) {
	if configured != "" {
		if !configured.Valid() {
			return "", fmt.Errorf("configured SAPIENT encoding %q is unsupported", configured)
		}
		return configured, nil
	}
	normalized := strings.ToLower(contentType)
	switch {
	case strings.Contains(normalized, "json"):
		return sapientcodec.EncodingJSON, nil
	case strings.Contains(normalized, "protobuf"), strings.Contains(normalized, "x-protobuf"):
		return sapientcodec.EncodingProtobuf, nil
	default:
		return "", fmt.Errorf("could not infer SAPIENT encoding from content type %q", contentType)
	}
}

func readLimited(reader io.Reader, maxBytes int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("read SAPIENT HTTP response: %w", err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("SAPIENT HTTP response exceeds max_response_bytes %d", maxBytes)
	}
	return data, nil
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

func uptimeSinceAt(startedAt time.Time, now time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return now.Sub(startedAt)
}

func entityAlreadyExists(err error) (string, bool) {
	var mutationErr *sapientprojector.MutationFailureError
	if !errors.As(err, &mutationErr) {
		return "", false
	}
	if mutationErr.Kind != sapientprojector.MutationCreate ||
		mutationErr.ErrorCode != graph.ErrorCodeEntityExists ||
		mutationErr.EntityID == "" {
		return "", false
	}
	return mutationErr.EntityID, true
}
