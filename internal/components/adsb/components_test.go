package adsb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/ownership"
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

	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	record := mustADSBSnapshotRecord(t, now)
	raw := NewRawSnapshotPayload("adsb:http:test", "https://example.test/states/all", now, http.StatusOK, record.RawJSON)
	rawWire := mustBaseMessageJSON(t, RawSnapshotType, raw, "semops-input-adsb-http", now)
	rawEnvelope, err := message.NewDecoder(registry).Decode(rawWire)
	if err != nil {
		t.Fatalf("decode raw payload: %v", err)
	}
	if _, ok := rawEnvelope.Payload().(*RawSnapshotPayload); !ok {
		t.Fatalf("raw payload type = %T, want *RawSnapshotPayload", rawEnvelope.Payload())
	}

	snapshot, err := record.Snapshot()
	if err != nil {
		t.Fatalf("parse fixture snapshot: %v", err)
	}
	decoded := NewDecodedSnapshotPayload(record, snapshot)
	decodedWire := mustBaseMessageJSON(t, DecodedSnapshotType, decoded, "semops-processor-adsb-decode", now)
	decodedEnvelope, err := message.NewDecoder(registry).Decode(decodedWire)
	if err != nil {
		t.Fatalf("decode decoded-snapshot payload: %v", err)
	}
	if _, ok := decodedEnvelope.Payload().(*DecodedSnapshotPayload); !ok {
		t.Fatalf("decoded payload type = %T, want *DecodedSnapshotPayload", decodedEnvelope.Payload())
	}
}

func TestADSBComponentsExposeHTTPTimerStreamAndGraphPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*HTTPPollerComponent)(nil)
	var _ component.DebugStatusProvider = (*HTTPPollerComponent)(nil)
	var _ component.LifecycleComponent = (*DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*ProjectorComponent)(nil)

	bus := &recordingBus{}
	poller, err := NewHTTPPollerComponent(HTTPPollerConfig{
		URL:           "https://opensky-network.org/api/states/all",
		PollInterval:  15 * time.Second,
		ContactPolicy: "semops-demo@example.invalid",
		AuthRef:       "opensky",
	}, bus)
	if err != nil {
		t.Fatalf("new poller: %v", err)
	}
	decoder, err := NewDecoderComponent(DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	projector, err := NewProjectorComponent(ProjectorConfig{Writer: &recordingPlanWriter{}}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	if poller.Meta().Type != "input" {
		t.Fatalf("poller component type = %q, want input", poller.Meta().Type)
	}
	if decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf(
			"processor component types = %q/%q, want processor/processor",
			decoder.Meta().Type,
			projector.Meta().Type,
		)
	}
	inputPorts := poller.InputPorts()
	if len(inputPorts) != 2 {
		t.Fatalf("poller input ports = %d, want HTTP client and timer", len(inputPorts))
	}
	httpPort, ok := inputPorts[0].Config.(component.HTTPClientPort)
	if !ok {
		t.Fatalf("poller adsb_feed config = %T, want HTTPClientPort", inputPorts[0].Config)
	}
	if got, want := httpPort.Type(), "http-client"; got != want {
		t.Fatalf("HTTP client port type = %q, want %q", got, want)
	}
	if got, want := httpPort.TriggerPort, "poll_tick"; got != want {
		t.Fatalf("HTTP client trigger = %q, want %q", got, want)
	}
	if got, want := httpPort.Interface.Compatible[0], RawSnapshotType.Key(); got != want {
		t.Fatalf("HTTP client interface compatible = %q, want %q", got, want)
	}
	if got, want := inputPorts[1].Config.Type(), "timer"; got != want {
		t.Fatalf("poll_tick config type = %q, want %q", got, want)
	}
	if got, want := poller.OutputPorts()[0].Config.(component.NATSPort).Subject, DefaultRawSubject; got != want {
		t.Fatalf("poller raw subject = %q, want %q", got, want)
	}
	if got, want := poller.ConfigSchema().Properties["stale_after"].Default, "1m0s"; got != want {
		t.Fatalf("poller stale_after default = %v, want %s", got, want)
	}
	if got, want := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject, DefaultDecodedSubject; got != want {
		t.Fatalf("decoder decoded subject = %q, want %q", got, want)
	}
	for _, port := range projector.OutputPorts() {
		if got, want := port.Config.Type(), "nats-request"; got != want {
			t.Fatalf("projector output port %q type = %q, want %q", port.Name, got, want)
		}
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(poller.Meta().Name, poller); err != nil {
		t.Fatalf("add poller: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add decoder: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add projector: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect flowgraph: %v", err)
	}
	pollerNode := fg.GetNodes()[poller.Meta().Name]
	if got, want := pollerNode.InputPorts[0].Pattern, flowgraph.PatternHTTPClient; got != want {
		t.Fatalf("HTTP polling input pattern = %q, want %q", got, want)
	}
	requireEdge(
		t,
		fg.GetEdges(),
		poller.Meta().Name,
		"raw_snapshots",
		decoder.Meta().Name,
		"raw_snapshots",
		DefaultRawSubject,
	)
	requireEdge(
		t,
		fg.GetEdges(),
		decoder.Meta().Name,
		"decoded_snapshots",
		projector.Meta().Name,
		"decoded_snapshots",
		DefaultDecodedSubject,
	)
	analysis := fg.AnalyzeConnectivity()
	for _, orphan := range analysis.OrphanedPorts {
		if orphan.ComponentName == poller.Meta().Name && orphan.PortName == "adsb_feed" {
			t.Fatalf("HTTP client input reported as orphaned: %+v", orphan)
		}
	}
}

func TestHTTPPollerPollOncePublishesRawBaseMessage(t *testing.T) {
	now := time.Date(2026, 6, 21, 10, 30, 0, 0, time.UTC)
	record := mustADSBSnapshotRecord(t, now)
	var userAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.UserAgent()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"fixture-1"`)
		w.Header().Set("Last-Modified", now.Format(http.TimeFormat))
		_, _ = w.Write(record.RawJSON)
	}))
	defer server.Close()

	bus := &recordingBus{}
	poller, err := NewHTTPPollerComponent(HTTPPollerConfig{
		Source:        "adsb:http:test",
		URL:           server.URL,
		Client:        server.Client(),
		ContactPolicy: "semops-test@example.invalid",
		Clock:         func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new poller: %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if userAgent != "semops-test@example.invalid" {
		t.Fatalf("user agent = %q", userAgent)
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
	payload, ok := envelope.Payload().(*RawSnapshotPayload)
	if !ok {
		t.Fatalf("payload = %T, want *RawSnapshotPayload", envelope.Payload())
	}
	if payload.Source != "adsb:http:test" || payload.Endpoint != server.URL {
		t.Fatalf("payload source/endpoint = %+v", payload)
	}
	if payload.StatusCode != http.StatusOK || payload.ETag != `"fixture-1"` {
		t.Fatalf("payload status/cache metadata = %+v", payload)
	}
	if string(payload.RawJSON) != string(record.RawJSON) {
		t.Fatalf("raw JSON changed across raw publish")
	}
	status := poller.DebugStatus().(HTTPPollerDebugStatus)
	if !status.LastFreshData.Equal(now) {
		t.Fatalf("last fresh data = %s, want %s", status.LastFreshData, now)
	}
	if !status.LastProviderContact.Equal(now) || status.LastStatusCode != http.StatusOK {
		t.Fatalf("provider contact status = %+v", status)
	}
}

func TestHTTPPollerAndDecoderCaptureProviderReplayFixture(t *testing.T) {
	now := time.Date(2026, 6, 21, 10, 45, 0, 0, time.UTC)
	record := mustADSBSnapshotRecord(t, now)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"fixture-provider-1"`)
		_, _ = w.Write(record.RawJSON)
	}))
	defer server.Close()

	bus := &recordingBus{}
	poller, err := NewHTTPPollerComponent(HTTPPollerConfig{
		Source: "adsb:http:provider-fixture",
		URL:    server.URL,
		Client: server.Client(),
		Clock:  func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new poller: %v", err)
	}

	replayPath := filepath.Join(t.TempDir(), "adsb-provider.jsonl")
	registry := payloadregistry.New()
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source:   "opensky-fixture",
		Registry: registry,
		Replay:   adsbcodec.NewReplayStore(replayPath),
		Clock:    func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("poll once: %v", err)
	}
	raw := bus.singlePublished(t, DefaultRawSubject)
	if err := decoder.HandleRawMessage(context.Background(), raw.data); err != nil {
		t.Fatalf("decode provider raw message: %v", err)
	}

	records, err := adsbcodec.LoadReplay(replayPath)
	if err != nil {
		t.Fatalf("load provider replay: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("provider replay records = %d, want 1", len(records))
	}
	if records[0].Source != "opensky-fixture" ||
		records[0].Ref != "adsb://raw/opensky-fixture/00000001" ||
		len(records[0].RawJSON) == 0 {
		t.Fatalf("provider replay record = %+v", records[0])
	}
	snapshot, err := records[0].Snapshot()
	if err != nil {
		t.Fatalf("parse provider replay raw JSON: %v", err)
	}
	if len(snapshot.States) != 2 {
		t.Fatalf("provider replay states = %d, want 2", len(snapshot.States))
	}
}

func TestHTTPPollerHealthReportsStaleWhenFreshDataAgesPastThreshold(t *testing.T) {
	now := time.Date(2026, 6, 21, 11, 0, 0, 0, time.UTC)
	current := now
	poller, err := NewHTTPPollerComponent(HTTPPollerConfig{
		URL:          "https://example.test/states/all",
		PollInterval: time.Hour,
		StaleAfter:   10 * time.Minute,
		Clock:        func() time.Time { return current },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new poller: %v", err)
	}
	if err := poller.Initialize(); err != nil {
		t.Fatalf("initialize poller: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := poller.Start(ctx); err != nil {
		t.Fatalf("start poller: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		if err := poller.Stop(time.Second); err != nil {
			t.Fatalf("stop poller: %v", err)
		}
	})

	health := poller.Health()
	if !health.Healthy || health.Status != "started" {
		t.Fatalf("initial health = %+v, want healthy started", health)
	}
	current = now.Add(11 * time.Minute)
	health = poller.Health()
	if health.Healthy || health.Status != "stale" {
		t.Fatalf("stale health = %+v, want unhealthy stale", health)
	}
	if !strings.Contains(health.LastError, "no fresh payload") {
		t.Fatalf("stale health error = %q", health.LastError)
	}
}

func TestDecoderConsumesRawAndPublishesDecodedSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 21, 11, 30, 0, 0, time.UTC)
	record := mustADSBSnapshotRecord(t, now)
	bus := &recordingBus{}
	registry := payloadregistry.New()
	replay := &recordingReplay{}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source:   "opensky-fixture",
		Clock:    func() time.Time { return now },
		Registry: registry,
		Replay:   replay,
	}, bus)
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}
	if err := decoder.Initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	raw := NewRawSnapshotPayload("adsb:http:test", "https://example.test/states/all", now, http.StatusOK, record.RawJSON)
	rawWire := mustBaseMessageJSON(t, RawSnapshotType, raw, "semops-input-adsb-http", now)
	if err := decoder.HandleRawMessage(context.Background(), rawWire); err != nil {
		t.Fatalf("handle raw: %v", err)
	}

	published := bus.singlePublished(t, DefaultDecodedSubject)
	envelope, err := message.NewDecoder(registry).Decode(published.data)
	if err != nil {
		t.Fatalf("decode published decoded message: %v", err)
	}
	payload, ok := envelope.Payload().(*DecodedSnapshotPayload)
	if !ok {
		t.Fatalf("payload = %T, want *DecodedSnapshotPayload", envelope.Payload())
	}
	if payload.RawRef == "" || payload.Source != "opensky-fixture" {
		t.Fatalf("decoded payload source/ref = %+v", payload)
	}
	if len(payload.Snapshot.States) != 2 {
		t.Fatalf("decoded states = %d, want 2", len(payload.Snapshot.States))
	}
	if len(replay.records) != 1 || replay.records[0].Ref != payload.RawRef {
		t.Fatalf("replay records = %+v, decoded raw ref = %q", replay.records, payload.RawRef)
	}
}

func TestDecoderCapturesMalformedSnapshotsBeforeParseFailure(t *testing.T) {
	now := time.Date(2026, 6, 21, 11, 45, 0, 0, time.UTC)
	replay := &recordingReplay{}
	decoder, err := NewDecoderComponent(DecoderConfig{
		Source: "opensky-fixture",
		Replay: replay,
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new decoder: %v", err)
	}

	payload := NewRawSnapshotPayload(
		"adsb:http:test",
		"https://example.test/states/all",
		now,
		http.StatusOK,
		[]byte(`{"states":[]}`),
	)
	err = decoder.HandleRawPayload(context.Background(), payload)
	if err == nil || !strings.Contains(err.Error(), "parse ADS-B snapshot") {
		t.Fatalf("malformed error = %v, want parse failure", err)
	}
	if len(replay.records) != 1 || replay.records[0].Ref != "adsb://raw/opensky-fixture/00000001" {
		t.Fatalf("malformed replay records = %+v", replay.records)
	}
}

func TestProjectorConsumesDecodedSnapshotAndWritesGraphPlan(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	record := mustADSBSnapshotRecord(t, now)
	snapshot, err := record.Snapshot()
	if err != nil {
		t.Fatalf("parse fixture snapshot: %v", err)
	}
	bus := &recordingBus{}
	registry := payloadregistry.New()
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Registry: registry,
		Projector: adsbprojector.NewProjector(adsbprojector.Config{
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerADSB: ownership.ExpectedOwnerToken(cop.OwnerADSB, "component-test"),
			},
			TraceID: "component-test",
		}),
		Writer: writer,
		Clock:  func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	if err := projector.Initialize(); err != nil {
		t.Fatalf("initialize projector: %v", err)
	}

	payload := NewDecodedSnapshotPayload(record, snapshot)
	wire := mustBaseMessageJSON(t, DecodedSnapshotType, payload, "semops-processor-adsb-decode", now)
	if err := projector.HandleDecodedMessage(context.Background(), wire); err != nil {
		t.Fatalf("handle decoded: %v", err)
	}

	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	if len(writer.plans[0].Mutations) != 2 ||
		writer.plans[0].Mutations[0].Kind != adsbprojector.MutationCreate ||
		writer.plans[0].Mutations[1].Kind != adsbprojector.MutationCreate {
		t.Fatalf("plan = %+v, want ADS-B track creates", writer.plans[0])
	}
	create := writer.plans[0].Mutations[0].Create
	if create.OwnerToken != "semops.feed.adsb#component-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "component-test" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
}

func TestProjectorReconcilesExistingBirth(t *testing.T) {
	now := time.Date(2026, 6, 21, 12, 15, 0, 0, time.UTC)
	record := mustADSBSnapshotRecord(t, now)
	snapshot, err := record.Snapshot()
	if err != nil {
		t.Fatalf("parse fixture snapshot: %v", err)
	}
	writer := &recordingPlanWriter{
		failures: []error{
			&adsbprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      adsbprojector.MutationCreate,
				EntityID:  adsbprojector.EntityID("c360", "edge", "a1b2c3"),
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
		},
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: adsbprojector.NewProjector(adsbprojector.Config{
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerADSB: ownership.ExpectedOwnerToken(cop.OwnerADSB, "component-test"),
			},
		}),
		Writer: writer,
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	payload := NewDecodedSnapshotPayload(record, snapshot)
	if err := projector.HandleDecodedPayload(context.Background(), payload); err != nil {
		t.Fatalf("handle decoded with birth reconciliation: %v", err)
	}
	if len(writer.plans) != 2 {
		t.Fatalf("plans = %d, want create retry then update/create", len(writer.plans))
	}
	last := writer.plans[len(writer.plans)-1]
	if len(last.Mutations) != 2 ||
		last.Mutations[0].Kind != adsbprojector.MutationUpdate ||
		last.Mutations[1].Kind != adsbprojector.MutationCreate {
		t.Fatalf("last plan = %+v, want update/create after reconciling existing birth", last)
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
	records []adsbcodec.RawSnapshotRecord
}

func (r *recordingReplay) Append(record adsbcodec.RawSnapshotRecord) error {
	r.records = append(r.records, record)
	return nil
}

type recordingPlanWriter struct {
	plans    []adsbprojector.Plan
	failures []error
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan adsbprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.failures) == 0 {
		return nil
	}
	err := w.failures[0]
	w.failures = w.failures[1:]
	return err
}

func mustADSBSnapshotRecord(t *testing.T, start time.Time) adsbcodec.RawSnapshotRecord {
	t.Helper()
	records, err := adsbcodec.OpenSkyFixtureRecords(start)
	if err != nil {
		t.Fatalf("load ADS-B fixture records: %v", err)
	}
	if len(records) == 0 {
		t.Fatalf("ADS-B fixture records empty")
	}
	return records[0]
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
