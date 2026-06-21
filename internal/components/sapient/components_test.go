package sapient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

func TestPayloadRegistryRoundTripsRawAndDecodedPayloads(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads should be idempotent: %v", err)
	}

	now := time.Date(2026, 6, 21, 13, 0, 0, 0, time.UTC)
	raw := NewRawMessagePayload(
		"sapient:http:test",
		"https://example.test/sapient",
		now,
		http.StatusOK,
		sapientcodec.EncodingJSON,
		[]byte(sampleTaskAckJSON),
	)
	rawWire := mustBaseMessageJSON(t, RawMessageType, raw, "semops-input-sapient-http", now)
	rawEnvelope, err := message.NewDecoder(registry).Decode(rawWire)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	if _, ok := rawEnvelope.Payload().(*RawMessagePayload); !ok {
		t.Fatalf("raw payload type = %T, want *RawMessagePayload", rawEnvelope.Payload())
	}

	msg, err := sapientcodec.ParseJSONMessage([]byte(sampleTaskAckJSON))
	if err != nil {
		t.Fatalf("parse sample SAPIENT message: %v", err)
	}
	record := sapientcodec.RawMessageRecord{
		Ref:        "sapient://raw/test/json/00000001",
		Source:     "test",
		ReceivedAt: now,
		Encoding:   sapientcodec.EncodingJSON,
		Content:    msg.Content,
		NodeID:     msg.NodeID,
		MessageAt:  msg.Timestamp,
		RawPayload: []byte(sampleTaskAckJSON),
	}
	decoded := NewDecodedMessagePayload(record, msg)
	decodedWire := mustBaseMessageJSON(t, DecodedMessageType, decoded, "semops-processor-sapient-decode", now)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decodedWire)
	if err != nil {
		t.Fatalf("decode decoded-message payload: %v", err)
	}
	if _, ok := decodedEnvelope.Payload().(*DecodedMessagePayload); !ok {
		t.Fatalf("decoded payload type = %T, want *DecodedMessagePayload", decodedEnvelope.Payload())
	}
}

func TestSAPIENTComponentsExposeHTTPAndStreamPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*HTTPInputComponent)(nil)
	var _ component.DebugStatusProvider = (*HTTPInputComponent)(nil)
	var _ component.LifecycleComponent = (*DecoderComponent)(nil)

	bus := &recordingBus{}
	input, err := NewHTTPInputComponent(HTTPInputConfig{
		URL:           "https://apex.example.invalid/sapient/messages",
		PollInterval:  15 * time.Second,
		ContactPolicy: "semops-demo@example.invalid",
		AuthRef:       "sapient-apex",
	}, bus)
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}

	if input.Meta().Type != "input" {
		t.Fatalf("input component type = %q, want input", input.Meta().Type)
	}
	if decoder.Meta().Type != "processor" {
		t.Fatalf("decoder component type = %q, want processor", decoder.Meta().Type)
	}
	inputPorts := input.InputPorts()
	if len(inputPorts) != 2 {
		t.Fatalf("input ports = %d, want HTTP client and timer", len(inputPorts))
	}
	httpPort, ok := inputPorts[0].Config.(component.HTTPClientPort)
	if !ok {
		t.Fatalf("input sapient_feed config = %T, want HTTPClientPort", inputPorts[0].Config)
	}
	if got, want := httpPort.Type(), "http-client"; got != want {
		t.Fatalf("HTTP client port type = %q, want %q", got, want)
	}
	if got, want := httpPort.TriggerPort, "poll_tick"; got != want {
		t.Fatalf("HTTP client trigger = %q, want %q", got, want)
	}
	if got, want := httpPort.Interface.Compatible[0], RawMessageType.Key(); got != want {
		t.Fatalf("HTTP client interface compatible = %q, want %q", got, want)
	}
	if got, want := inputPorts[1].Config.Type(), "timer"; got != want {
		t.Fatalf("poll_tick config type = %q, want %q", got, want)
	}
	if got, want := input.OutputPorts()[0].Config.(component.NATSPort).Subject, DefaultRawSubject; got != want {
		t.Fatalf("input raw subject = %q, want %q", got, want)
	}
	if got, want := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject, DefaultDecodedSubject; got != want {
		t.Fatalf("decoder decoded subject = %q, want %q", got, want)
	}
	if len(decoder.OutputPorts()) != 1 || len(input.OutputPorts()) != 1 {
		t.Fatalf("SAPIENT preflight components should not expose graph request ports")
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add input: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add decoder: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect flowgraph: %v", err)
	}
	inputNode := fg.GetNodes()[input.Meta().Name]
	if got, want := inputNode.InputPorts[0].Pattern, flowgraph.PatternHTTPClient; got != want {
		t.Fatalf("HTTP input pattern = %q, want %q", got, want)
	}
	requireEdge(
		t,
		fg.GetEdges(),
		input.Meta().Name,
		"raw_messages",
		decoder.Meta().Name,
		"raw_messages",
		DefaultRawSubject,
	)
	analysis := fg.AnalyzeConnectivity()
	for _, orphan := range analysis.OrphanedPorts {
		if orphan.ComponentName == input.Meta().Name && orphan.PortName == "sapient_feed" {
			t.Fatalf("HTTP client input reported as orphaned: %+v", orphan)
		}
	}
}

func TestHTTPInputPollOncePublishesRawBaseMessage(t *testing.T) {
	now := time.Date(2026, 6, 21, 13, 30, 0, 0, time.UTC)
	client := &fakeHTTPClient{
		resp: httpResponse(http.StatusOK, "application/json", []byte(sampleTaskAckJSON)),
	}
	bus := &recordingBus{}
	input, err := NewHTTPInputComponent(HTTPInputConfig{
		Source:        "sapient:http:test",
		URL:           "https://apex.example.invalid/sapient/messages",
		Client:        client,
		ContactPolicy: "semops-test@example.invalid",
		Clock:         func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	if err := input.PollOnce(context.Background()); err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if client.req.UserAgent() != "semops-test@example.invalid" {
		t.Fatalf("user agent = %q", client.req.UserAgent())
	}

	published := bus.singlePublished(t, DefaultRawSubject)
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published raw message: %v", err)
	}
	payload, ok := envelope.Payload().(*RawMessagePayload)
	if !ok {
		t.Fatalf("payload = %T, want *RawMessagePayload", envelope.Payload())
	}
	if payload.Source != "sapient:http:test" || payload.Endpoint != "https://apex.example.invalid/sapient/messages" {
		t.Fatalf("payload source/endpoint = %+v", payload)
	}
	if payload.StatusCode != http.StatusOK || payload.Encoding != sapientcodec.EncodingJSON {
		t.Fatalf("payload status/encoding = %+v", payload)
	}
	status := input.DebugStatus().(HTTPInputDebugStatus)
	if !status.LastFreshData.Equal(now) || status.LastContentType != "application/json" {
		t.Fatalf("debug status = %+v", status)
	}
}

func TestHTTPInputRequiresKnownEncodingWhenContentTypeIsAmbiguous(t *testing.T) {
	now := time.Date(2026, 6, 21, 13, 45, 0, 0, time.UTC)
	input, err := NewHTTPInputComponent(HTTPInputConfig{
		URL:    "https://apex.example.invalid/sapient/messages",
		Client: &fakeHTTPClient{resp: httpResponse(http.StatusOK, "application/octet-stream", []byte(sampleTaskAckJSON))},
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	err = input.PollOnce(context.Background())
	if err == nil || !strings.Contains(err.Error(), "could not infer SAPIENT encoding") {
		t.Fatalf("ambiguous encoding error = %v", err)
	}
}

func TestHTTPInputAndDecoderCaptureProviderReplayFixture(t *testing.T) {
	now := time.Date(2026, 6, 21, 14, 0, 0, 0, time.UTC)
	bus := &recordingBus{}
	input, err := NewHTTPInputComponent(HTTPInputConfig{
		Source: "sapient:http:provider-fixture",
		URL:    "https://apex.example.invalid/sapient/messages",
		Client: &fakeHTTPClient{resp: httpResponse(http.StatusOK, "application/json", []byte(sampleTaskAckJSON))},
		Clock:  func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new input: %v", err)
	}

	replayPath := filepath.Join(t.TempDir(), "sapient-provider.jsonl")
	registry := payloadregistry.New()
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source:   "sapient-fixture",
		Registry: registry,
		Replay:   sapientcodec.NewReplayStore(replayPath),
		Clock:    func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	if err := input.PollOnce(context.Background()); err != nil {
		t.Fatalf("poll once: %v", err)
	}
	raw := bus.singlePublished(t, DefaultRawSubject)
	if err := decoder.HandleRawMessage(context.Background(), raw.data); err != nil {
		t.Fatalf("decode provider raw message: %v", err)
	}

	records, err := sapientcodec.LoadReplay(replayPath)
	if err != nil {
		t.Fatalf("load provider replay: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("provider replay records = %d, want 1", len(records))
	}
	if records[0].Source != "sapient-fixture" ||
		records[0].Ref != "sapient://raw/sapient-fixture/json/00000001" ||
		records[0].Content != sapientcodec.ContentTaskAck {
		t.Fatalf("provider replay record = %+v", records[0])
	}
	if _, err := records[0].Message(nil); err != nil {
		t.Fatalf("parse provider replay raw payload: %v", err)
	}
}

func TestDecoderConsumesRawAndPublishesDecodedMessage(t *testing.T) {
	now := time.Date(2026, 6, 21, 14, 30, 0, 0, time.UTC)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	replay := &recordingReplay{}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source:   "sapient-fixture",
		Registry: registry,
		Replay:   replay,
		Clock:    func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	raw := NewRawMessagePayload(
		"sapient:http:test",
		"https://example.test/sapient",
		now,
		http.StatusOK,
		sapientcodec.EncodingJSON,
		[]byte(sampleTaskAckJSON),
	)
	rawWire := mustBaseMessageJSON(t, RawMessageType, raw, "semops-input-sapient-http", now)
	if err := decoder.HandleRawMessage(context.Background(), rawWire); err != nil {
		t.Fatalf("handle raw: %v", err)
	}

	published := bus.singlePublished(t, DefaultDecodedSubject)
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published decoded message: %v", err)
	}
	payload, ok := envelope.Payload().(*DecodedMessagePayload)
	if !ok {
		t.Fatalf("payload = %T, want *DecodedMessagePayload", envelope.Payload())
	}
	if payload.Content != sapientcodec.ContentTaskAck || payload.NodeID == "" || payload.RawRef == "" {
		t.Fatalf("decoded payload = %+v", payload)
	}
	if len(replay.records) != 1 || replay.records[0].Ref != payload.RawRef {
		t.Fatalf("replay records = %+v, decoded raw ref = %q", replay.records, payload.RawRef)
	}
}

func TestDecoderCapturesMalformedMessagesBeforeParseFailure(t *testing.T) {
	now := time.Date(2026, 6, 21, 14, 45, 0, 0, time.UTC)
	replay := &recordingReplay{}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source: "sapient-fixture",
		Replay: replay,
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}

	payload := NewRawMessagePayload(
		"sapient:http:test",
		"https://example.test/sapient",
		now,
		http.StatusOK,
		sapientcodec.EncodingJSON,
		[]byte(`{"nodeId":"not-a-uuid"}`),
	)
	err = decoder.HandleRawPayload(context.Background(), payload)
	if err == nil || !strings.Contains(err.Error(), "parse SAPIENT payload") {
		t.Fatalf("malformed error = %v, want parse failure", err)
	}
	if len(replay.records) != 1 || replay.records[0].Ref != "sapient://raw/sapient-fixture/json/00000001" {
		t.Fatalf("malformed replay records = %+v", replay.records)
	}
}

func TestHTTPInputHealthReportsStaleWhenFreshDataAgesPastThreshold(t *testing.T) {
	now := time.Date(2026, 6, 21, 15, 0, 0, 0, time.UTC)
	current := now
	input, err := NewHTTPInputComponent(HTTPInputConfig{
		URL:          "https://example.test/sapient",
		PollInterval: time.Hour,
		StaleAfter:   10 * time.Minute,
		Clock:        func() time.Time { return current },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new input: %v", err)
	}
	if err := input.Initialize(); err != nil {
		t.Fatalf("initialize input: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := input.Start(ctx); err != nil {
		t.Fatalf("start input: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		if err := input.Stop(time.Second); err != nil {
			t.Fatalf("stop input: %v", err)
		}
	})

	health := input.Health()
	if !health.Healthy || health.Status != "started" {
		t.Fatalf("initial health = %+v, want healthy started", health)
	}
	current = now.Add(11 * time.Minute)
	health = input.Health()
	if health.Healthy || health.Status != "stale" {
		t.Fatalf("stale health = %+v, want unhealthy stale", health)
	}
	if !strings.Contains(health.LastError, "no fresh payload") {
		t.Fatalf("stale health error = %q", health.LastError)
	}
}

type fakeHTTPClient struct {
	resp *http.Response
	req  *http.Request
	err  error
}

func (c *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.req = req.Clone(req.Context())
	if c.err != nil {
		return nil, c.err
	}
	return c.resp, nil
}

func httpResponse(status int, contentType string, body []byte) *http.Response {
	header := http.Header{}
	header.Set("Content-Type", contentType)
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

type publishedMessage struct {
	subject string
	data    []byte
}

type recordingBus struct {
	published     []publishedMessage
	subscriptions []string
}

func (b *recordingBus) Publish(_ context.Context, subject string, data []byte) error {
	b.published = append(b.published, publishedMessage{
		subject: subject,
		data:    append([]byte(nil), data...),
	})
	return nil
}

func (b *recordingBus) Subscribe(
	_ context.Context,
	subject string,
	_ func(context.Context, *nats.Msg),
) (Subscription, error) {
	b.subscriptions = append(b.subscriptions, subject)
	return fakeSubscription{}, nil
}

func (b *recordingBus) singlePublished(t *testing.T, subject string) publishedMessage {
	t.Helper()
	var matches []publishedMessage
	for _, msg := range b.published {
		if msg.subject == subject {
			matches = append(matches, msg)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("published messages for %s = %d, want 1; all=%+v", subject, len(matches), b.published)
	}
	return matches[0]
}

type fakeSubscription struct{}

func (fakeSubscription) Unsubscribe() error { return nil }

type recordingReplay struct {
	records []sapientcodec.RawMessageRecord
}

func (r *recordingReplay) Append(record sapientcodec.RawMessageRecord) error {
	r.records = append(r.records, record)
	return nil
}

func mustBaseMessageJSON(
	t *testing.T,
	msgType message.Type,
	payload message.Payload,
	source string,
	observedAt time.Time,
) []byte {
	t.Helper()
	data, err := marshalBaseMessage(msgType, payload, source, observedAt)
	if err != nil {
		t.Fatalf("marshal base message: %v", err)
	}
	return data
}

func requireEdge(
	t *testing.T,
	edges []flowgraph.FlowEdge,
	fromComponent string,
	fromPort string,
	toComponent string,
	toPort string,
	connectionID string,
) {
	t.Helper()
	for _, edge := range edges {
		if edge.From.ComponentName == fromComponent &&
			edge.From.PortName == fromPort &&
			edge.To.ComponentName == toComponent &&
			edge.To.PortName == toPort &&
			edge.Pattern == flowgraph.PatternStream &&
			edge.ConnectionID == connectionID {
			return
		}
	}
	t.Fatalf("missing edge %s.%s -> %s.%s (%s) in %+v", fromComponent, fromPort, toComponent, toPort, connectionID, edges)
}

const sampleTaskAckJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "taskAck": {
    "taskId": "01H4R63D7NVN8444Z5M77WEBY8",
    "taskStatus": "TASK_STATUS_ACCEPTED",
    "reason": ["accepted for preflight"]
  }
}`
