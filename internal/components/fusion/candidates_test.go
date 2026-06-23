package fusion

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/nats-io/nats.go"
)

func TestCandidateProducerExposesTimerGraphAndCandidatePorts(t *testing.T) {
	var _ component.LifecycleComponent = (*CandidateProducerComponent)(nil)

	requester := &recordingPrefixRequester{}
	producer, err := NewCandidateProducerComponent(CandidateProducerConfig{
		Requester: requester,
	}, &publishingBus{})
	if err != nil {
		t.Fatalf("new candidate producer: %v", err)
	}
	projector, err := NewProjectorComponent(ProjectorConfig{Writer: &recordingPlanWriter{}}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	if producer.Meta().Type != "processor" {
		t.Fatalf("producer type = %q, want processor", producer.Meta().Type)
	}
	inputs := producer.InputPorts()
	if len(inputs) != 1 || inputs[0].Config.Type() != "timer" {
		t.Fatalf("producer inputs = %+v, want timer trigger", inputs)
	}
	outputs := producer.OutputPorts()
	if len(outputs) != 2 {
		t.Fatalf("producer outputs = %d, want graph request and candidate stream", len(outputs))
	}
	if got, want := outputs[0].Config.(component.NATSRequestPort).Subject, SubjectGraphQueryPrefix; got != want {
		t.Fatalf("graph query subject = %q, want %q", got, want)
	}
	if got, want := outputs[1].Config.(component.NATSPort).Subject, DefaultCandidateSubject; got != want {
		t.Fatalf("candidate subject = %q, want %q", got, want)
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(producer.Meta().Name, producer); err != nil {
		t.Fatalf("add producer: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add projector: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect flowgraph: %v", err)
	}
	requireEdge(
		t,
		fg.GetEdges(),
		producer.Meta().Name,
		"track_candidates",
		projector.Meta().Name,
		"track_candidates",
		DefaultCandidateSubject,
	)
}

func TestCandidateProducerPublishesSourcePairBatchesFromGraphTracks(t *testing.T) {
	now := time.Date(2026, 6, 23, 22, 0, 0, 0, time.UTC)
	requester := &recordingPrefixRequester{
		responses: map[string][]graph.EntityState{
			"c360.edge.cop.mavlink.track": {
				trackEntity(
					"c360.edge.cop.mavlink.track.system-42",
					"mavlink",
					"mavlink.system.42",
					"POINT(-77.0002000 38.9001000)",
					now.Add(-2*time.Second),
					1,
					"mavlink://raw/udp/0001",
				),
			},
			"c360.edge.cop.adsb.track": {
				trackEntity(
					"c360.edge.cop.adsb.track.a1b2c3",
					"adsb",
					"adsb.icao24.a1b2c3",
					"POINT(-77.0000500 38.9002800)",
					now,
					0.88,
					"adsb://opensky/state/0001",
				),
			},
		},
	}
	bus := &publishingBus{}
	producer, err := NewCandidateProducerComponent(CandidateProducerConfig{
		CandidateSubject:   "semops.fusion.test_candidates",
		Requester:          requester,
		Sources:            []CandidateSourceScope{{Org: "c360", Platform: "edge", Source: "mavlink"}, {Org: "c360", Platform: "edge", Source: "adsb"}},
		MaxPairComparisons: 1,
		Clock:              func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new candidate producer: %v", err)
	}

	if err := producer.ScanAndPublish(context.Background()); err != nil {
		t.Fatalf("scan and publish candidates: %v", err)
	}

	if len(requester.requests) != 2 {
		t.Fatalf("prefix requests = %+v, want mavlink and adsb", requester.requests)
	}
	if requester.requests[0].Prefix != "c360.edge.cop.mavlink.track" ||
		requester.requests[1].Prefix != "c360.edge.cop.adsb.track" {
		t.Fatalf("prefix request order = %+v", requester.requests)
	}
	if len(bus.published) != 1 {
		t.Fatalf("published batches = %d, want 1", len(bus.published))
	}
	if bus.published[0].subject != "semops.fusion.test_candidates" {
		t.Fatalf("published subject = %q", bus.published[0].subject)
	}
	payload := decodeCandidatePayload(t, bus.published[0].data)
	if payload.Source != "semops-processor-fusion-candidates" {
		t.Fatalf("payload source = %q", payload.Source)
	}
	if len(payload.Primary) != 1 || len(payload.Candidates) != 1 {
		t.Fatalf("payload tracks = primary %d candidates %d", len(payload.Primary), len(payload.Candidates))
	}
	if payload.Primary[0].Source != "mavlink" || payload.Candidates[0].Source != "adsb" {
		t.Fatalf("payload source order = %+v / %+v", payload.Primary, payload.Candidates)
	}
	if payload.Primary[0].ID == payload.Candidates[0].ID {
		t.Fatalf("candidate producer emitted self-pair: %+v", payload)
	}
	if got := producer.DataFlow().LastActivity; !got.Equal(now) {
		t.Fatalf("last activity = %s, want %s", got, now)
	}
}

func TestCandidateProducerBoundsPairComparisonsAndBatchCount(t *testing.T) {
	now := time.Date(2026, 6, 23, 22, 30, 0, 0, time.UTC)
	requester := &recordingPrefixRequester{
		responses: map[string][]graph.EntityState{
			"c360.edge.cop.mavlink.track": {
				testTrack("c360.edge.cop.mavlink.track.system-42", "mavlink", now),
				testTrack("c360.edge.cop.mavlink.track.system-43", "mavlink", now),
			},
			"c360.edge.cop.adsb.track": {
				testTrack("c360.edge.cop.adsb.track.a1b2c3", "adsb", now),
				testTrack("c360.edge.cop.adsb.track.d4e5f6", "adsb", now),
			},
		},
	}
	bus := &publishingBus{}
	producer, err := NewCandidateProducerComponent(CandidateProducerConfig{
		Requester:          requester,
		Sources:            []CandidateSourceScope{{Org: "c360", Platform: "edge", Source: "mavlink"}, {Org: "c360", Platform: "edge", Source: "adsb"}},
		MaxPairComparisons: 1,
		MaxBatches:         2,
		Clock:              func() time.Time { return now },
	}, bus)
	if err != nil {
		t.Fatalf("new candidate producer: %v", err)
	}

	if err := producer.ScanAndPublish(context.Background()); err != nil {
		t.Fatalf("scan and publish candidates: %v", err)
	}
	if len(bus.published) != 2 {
		t.Fatalf("published batches = %d, want max batch cap 2", len(bus.published))
	}
	for _, msg := range bus.published {
		payload := decodeCandidatePayload(t, msg.data)
		if len(payload.Primary)*len(payload.Candidates) > 1 {
			t.Fatalf("batch exceeded comparison cap: %+v", payload)
		}
	}
}

func TestTrackObservationFromEntityRejectsMissingPosition(t *testing.T) {
	now := time.Date(2026, 6, 23, 23, 0, 0, 0, time.UTC)
	entity := graph.EntityState{
		ID:        "c360.edge.cop.mavlink.track.system-42",
		UpdatedAt: now,
		Triples: []message.Triple{
			testTriple("c360.edge.cop.mavlink.track.system-42", cop.TrackObservedAt, now, now, 1),
		},
	}
	if _, ok := trackObservationFromEntity(entity, CandidateSourceScope{Source: "mavlink"}, func() time.Time { return now }); ok {
		t.Fatal("track without position should not become a fusion candidate")
	}
}

type prefixQueryRecord struct {
	subject string
	timeout time.Duration
	graph.PrefixQueryRequest
}

type recordingPrefixRequester struct {
	requests  []prefixQueryRecord
	responses map[string][]graph.EntityState
}

func (r *recordingPrefixRequester) Request(
	_ context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	var request graph.PrefixQueryRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, err
	}
	r.requests = append(r.requests, prefixQueryRecord{
		subject:            subject,
		timeout:            timeout,
		PrefixQueryRequest: request,
	})
	return json.Marshal(graph.PrefixQueryResponse{Entities: r.responses[request.Prefix]})
}

type publishedCandidateMessage struct {
	subject string
	data    []byte
}

type publishingBus struct {
	published []publishedCandidateMessage
}

func (b *publishingBus) Publish(_ context.Context, subject string, data []byte) error {
	b.published = append(b.published, publishedCandidateMessage{
		subject: subject,
		data:    append([]byte(nil), data...),
	})
	return nil
}

func (b *publishingBus) Subscribe(
	context.Context,
	string,
	func(context.Context, *nats.Msg),
) (Subscription, error) {
	return recordingSubscription{}, nil
}

func decodeCandidatePayload(t *testing.T, data []byte) *CandidateBatchPayload {
	t.Helper()
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register fusion payloads: %v", err)
	}
	envelope, err := message.NewDecoder(registry).Decode(data)
	if err != nil {
		t.Fatalf("decode candidate payload: %v", err)
	}
	payload, ok := envelope.Payload().(*CandidateBatchPayload)
	if !ok {
		t.Fatalf("payload = %T, want *CandidateBatchPayload", envelope.Payload())
	}
	return payload
}

func testTrack(id string, source string, observedAt time.Time) graph.EntityState {
	return trackEntity(
		id,
		source,
		id+".native",
		"POINT(-77.0000000 38.9000000)",
		observedAt,
		0.9,
		source+"://fixture/test",
	)
}

func trackEntity(
	id string,
	source string,
	nativeID string,
	position string,
	observedAt time.Time,
	confidence float64,
	sourceRef string,
) graph.EntityState {
	return graph.EntityState{
		ID:        id,
		UpdatedAt: observedAt,
		Triples: []message.Triple{
			testTriple(id, cop.TrackNativeID, nativeID, observedAt, confidence),
			testTriple(id, cop.TrackObservedAt, observedAt, observedAt, confidence),
			testTriple(id, cop.TrackPosition, position, observedAt, confidence),
			testTriple(id, cop.ProvenanceSource, source, observedAt, confidence),
			testTriple(id, cop.ProvenanceConfidence, confidence, observedAt, confidence),
			testTriple(id, cop.ProvenanceObservedAt, observedAt, observedAt, confidence),
			testTriple(id, cop.ProvenanceSourceRef, sourceRef, observedAt, confidence),
		},
	}
}

func testTriple(subject string, predicate string, object any, observedAt time.Time, confidence float64) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "candidate-producer-test",
		Timestamp:  observedAt,
		Confidence: confidence,
	}
}

func requireCandidatePair(
	t *testing.T,
	payload *CandidateBatchPayload,
	wantPrimary string,
	wantCandidate string,
) {
	t.Helper()
	if len(payload.Primary) == 0 || len(payload.Candidates) == 0 {
		t.Fatalf("payload has no candidates: %+v", payload)
	}
	got := fusionassociation.EntityID("c360", "edge", payload.Primary[0].ID, payload.Candidates[0].ID)
	want := fusionassociation.EntityID("c360", "edge", wantPrimary, wantCandidate)
	if got != want {
		t.Fatalf("candidate pair = %q, want %q", got, want)
	}
}
