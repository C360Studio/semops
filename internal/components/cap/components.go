package cap

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

	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

const (
	DefaultHTTPPollURL          = "https://api.weather.gov/alerts/active"
	DefaultHTTPPollInterval     = 5 * time.Minute
	DefaultHTTPMaxResponseBytes = 4 * 1024 * 1024
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
		return errors.New("cap NATS bus requires a client")
	}
	return b.Client.Publish(ctx, subject, data)
}

func (b NATSBus) Subscribe(
	ctx context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.Client == nil {
		return nil, errors.New("cap NATS bus requires a client")
	}
	return b.Client.Subscribe(ctx, subject, handler)
}

type ReplayAppender interface {
	Append(record capcodec.RawAlertRecord) error
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPPollerConfig struct {
	Name             string
	Source           string
	URL              string
	Method           string
	RawSubject       string
	PollInterval     time.Duration
	ContactPolicy    string
	AuthRef          string
	MaxResponseBytes int
	ETag             string
	LastModified     string
	Client           HTTPClient
	Clock            func() time.Time
}

type HTTPPollerComponent struct {
	cfg HTTPPollerConfig
	bus Bus

	mu      sync.Mutex
	state   component.State
	cancel  context.CancelFunc
	done    chan error
	metrics flowCounters
}

func NewHTTPPollerComponent(cfg HTTPPollerConfig, bus Bus) (*HTTPPollerComponent, error) {
	if cfg.Name == "" {
		cfg.Name = "semops-input-cap-http"
	}
	if cfg.Source == "" {
		cfg.Source = "cap:http"
	}
	if cfg.URL == "" {
		cfg.URL = DefaultHTTPPollURL
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
		return nil, fmt.Errorf("cap HTTP poll interval must be greater than zero")
	}
	if cfg.MaxResponseBytes == 0 {
		cfg.MaxResponseBytes = DefaultHTTPMaxResponseBytes
	}
	if cfg.MaxResponseBytes < 0 {
		return nil, fmt.Errorf("cap HTTP max response bytes must be greater than zero")
	}
	if cfg.Client == nil {
		cfg.Client = http.DefaultClient
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if bus == nil {
		return nil, fmt.Errorf("cap HTTP poller component requires a bus")
	}
	return &HTTPPollerComponent{cfg: cfg, bus: bus, state: component.StateCreated}, nil
}

func (c *HTTPPollerComponent) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == component.StateCreated {
		c.state = component.StateInitialized
	}
	return nil
}

func (c *HTTPPollerComponent) Start(ctx context.Context) error {
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

func (c *HTTPPollerComponent) Stop(timeout time.Duration) error {
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
	return waitDone(done, timeout, "stop CAP HTTP poller")
}

func (c *HTTPPollerComponent) run(ctx context.Context) error {
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

func (c *HTTPPollerComponent) PollOnce(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.Lock()
	etag := c.cfg.ETag
	lastModified := c.cfg.LastModified
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, c.cfg.Method, c.cfg.URL, nil)
	if err != nil {
		return fmt.Errorf("build CAP HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/cap+xml, application/xml;q=0.9, text/xml;q=0.8")
	if c.cfg.ContactPolicy != "" {
		req.Header.Set("User-Agent", c.cfg.ContactPolicy)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := c.cfg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("poll CAP HTTP feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("poll CAP HTTP feed returned status %d", resp.StatusCode)
	}
	body, err := readLimited(resp.Body, c.cfg.MaxResponseBytes)
	if err != nil {
		return err
	}
	now := c.cfg.Clock().UTC()
	payload := NewRawAlertPayload(c.cfg.Source, c.cfg.URL, now, resp.StatusCode, body)
	payload.ETag = resp.Header.Get("ETag")
	payload.LastModified = resp.Header.Get("Last-Modified")
	if err := payload.Validate(); err != nil {
		return err
	}
	data, err := marshalBaseMessage(RawAlertType, payload, c.cfg.Name, now)
	if err != nil {
		return err
	}
	if err := c.bus.Publish(ctx, c.cfg.RawSubject, data); err != nil {
		return err
	}

	c.mu.Lock()
	c.cfg.ETag = payload.ETag
	c.cfg.LastModified = payload.LastModified
	c.mu.Unlock()
	c.recordMessage(len(body), now)
	return nil
}

func (c *HTTPPollerComponent) Meta() component.Metadata {
	return component.Metadata{
		Name:        c.cfg.Name,
		Type:        "input",
		Description: "CAP HTTP polling input component",
		Version:     "v0.1.0",
	}
}

func (c *HTTPPollerComponent) InputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "cap_feed",
			Direction:   component.DirectionInput,
			Required:    true,
			Description: "Outbound CAP HTTP feed polling dependency",
			Config: component.HTTPClientPort{
				Method:        c.cfg.Method,
				URLPattern:    c.cfg.URL,
				TriggerPort:   "poll_tick",
				AuthRef:       c.cfg.AuthRef,
				ContactPolicy: c.cfg.ContactPolicy,
				Interface: &component.InterfaceContract{
					Type:       "message.BaseMessage",
					Version:    "v1",
					Compatible: []string{RawAlertType.Key()},
				},
			},
		},
		{
			Name:        "poll_tick",
			Direction:   component.DirectionInput,
			Required:    true,
			Description: "Timer cadence for CAP HTTP polling",
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

func (c *HTTPPollerComponent) OutputPorts() []component.Port {
	return []component.Port{streamPort("raw_alerts", component.DirectionOutput, c.cfg.RawSubject, RawAlertType)}
}

func (c *HTTPPollerComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"url":                stringProperty("CAP HTTP endpoint URL", c.cfg.URL),
			"method":             stringProperty("HTTP method for CAP polling", c.cfg.Method),
			"raw_subject":        stringProperty("SemStreams subject carrying raw CAP alerts", c.cfg.RawSubject),
			"source":             stringProperty("Source label recorded in raw CAP payloads", c.cfg.Source),
			"poll_interval":      stringProperty("Timer cadence for CAP HTTP polling", c.cfg.PollInterval.String()),
			"contact_policy":     stringProperty("Public User-Agent/contact identity for feed providers", c.cfg.ContactPolicy),
			"auth_ref":           stringProperty("Secret reference for authenticated CAP feeds", c.cfg.AuthRef),
			"max_response_bytes": intProperty("Maximum accepted CAP HTTP response size", c.cfg.MaxResponseBytes),
		},
		Required: []string{"url", "method", "raw_subject", "source", "poll_interval"},
	}
}

func (c *HTTPPollerComponent) Health() component.HealthStatus {
	return c.metrics.health(c.state)
}

func (c *HTTPPollerComponent) DataFlow() component.FlowMetrics {
	return c.metrics.flow()
}

type DecoderConfig struct {
	Name           string
	Source         string
	RawSubject     string
	DecodedSubject string
	Registry       *payloadregistry.Registry
	Replay         ReplayAppender
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
		cfg.Name = "semops-processor-cap-decode"
	}
	if cfg.Source == "" {
		cfg.Source = "cap:decoder"
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
		return nil, fmt.Errorf("cap decoder component requires a bus")
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
		return fmt.Errorf("subscribe CAP decoder raw subject: %w", err)
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
		return fmt.Errorf("decode CAP raw BaseMessage: %w", err)
	}
	payload, ok := envelope.Payload().(*RawAlertPayload)
	if !ok {
		return fmt.Errorf("CAP decoder received payload %T, want *RawAlertPayload", envelope.Payload())
	}
	return c.HandleRawPayload(ctx, payload)
}

func (c *DecoderComponent) HandleRawPayload(ctx context.Context, payload *RawAlertPayload) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	alert, err := capcodec.Parse(payload.RawXML)
	if err != nil {
		return fmt.Errorf("parse CAP alert: %w", err)
	}
	record := rawAlertRecord(payload, alert)
	if err := c.appendReplay(record); err != nil {
		return err
	}
	decoded := NewDecodedAlertPayload(c.cfg.Source, record, alert)
	data, err := marshalBaseMessage(DecodedAlertType, decoded, c.cfg.Name, record.ReceivedAt)
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
		Description: "CAP raw-alert decoder processor",
		Version:     "v0.1.0",
	}
}

func (c *DecoderComponent) InputPorts() []component.Port {
	return []component.Port{streamPort("raw_alerts", component.DirectionInput, c.cfg.RawSubject, RawAlertType)}
}

func (c *DecoderComponent) OutputPorts() []component.Port {
	return []component.Port{
		streamPort("decoded_alerts", component.DirectionOutput, c.cfg.DecodedSubject, DecodedAlertType),
	}
}

func (c *DecoderComponent) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_subject":     stringProperty("SemStreams subject carrying raw CAP alerts", c.cfg.RawSubject),
			"decoded_subject": stringProperty("SemStreams subject carrying decoded CAP alerts", c.cfg.DecodedSubject),
			"source":          stringProperty("Source label recorded in decoded CAP payloads", c.cfg.Source),
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

func (c *DecoderComponent) appendReplay(record capcodec.RawAlertRecord) error {
	if c.cfg.Replay == nil {
		return nil
	}
	if err := c.cfg.Replay.Append(record); err != nil {
		return fmt.Errorf("append CAP replay record %q: %w", record.Ref, err)
	}
	return nil
}

func (c *DecoderComponent) markFailed(err error) {
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

func (c *HTTPPollerComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *HTTPPollerComponent) recordError(err error) {
	c.metrics.recordError(err)
}

func (c *DecoderComponent) recordMessage(size int, now time.Time) {
	c.metrics.recordMessage(size, now)
}

func (c *DecoderComponent) recordError(err error) {
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

func rawAlertRecord(payload *RawAlertPayload, alert capcodec.Alert) capcodec.RawAlertRecord {
	receivedAt := payload.ReceivedAt.UTC()
	return capcodec.RawAlertRecord{
		Ref:        rawAlertRef(payload.Source, alert.Identifier, receivedAt),
		Source:     payload.Source,
		ReceivedAt: receivedAt,
		Identifier: alert.Identifier,
		MsgType:    alert.MsgType,
		SentAt:     alert.Sent.UTC(),
		RawXML:     append([]byte(nil), payload.RawXML...),
	}
}

func rawAlertRef(source, identifier string, receivedAt time.Time) string {
	return fmt.Sprintf(
		"cap://raw/%s/%s/%d",
		sanitizeRefPart(source),
		sanitizeRefPart(identifier),
		receivedAt.UnixNano(),
	)
}

func sanitizeRefPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}

func readLimited(reader io.Reader, maxBytes int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("read CAP HTTP response: %w", err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("CAP HTTP response exceeds max_response_bytes %d", maxBytes)
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

func uptimeSince(startedAt time.Time) time.Duration {
	if startedAt.IsZero() {
		return 0
	}
	return time.Since(startedAt)
}
