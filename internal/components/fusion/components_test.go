package fusion

import (
	"context"
	"testing"
	"time"

	fusionassociation "github.com/c360studio/semops/internal/fusion/association"
	fusionprojector "github.com/c360studio/semops/internal/projectors/fusion"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/nats-io/nats.go"
)

func TestPayloadRegistryRoundTripsCandidateBatch(t *testing.T) {
	registry := payloadregistry.New()
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads: %v", err)
	}
	if err := RegisterPayloads(registry); err != nil {
		t.Fatalf("register payloads should be idempotent: %v", err)
	}

	now := time.Date(2026, 6, 23, 19, 0, 0, 0, time.UTC)
	payload := sampleCandidateBatch(now)
	wire := mustBaseMessageJSON(t, CandidateBatchType, payload, "semops-processor-track-candidates", now)
	envelope, err := message.NewDecoder(registry).Decode(wire)
	if err != nil {
		t.Fatalf("decode candidate batch: %v", err)
	}
	got, ok := envelope.Payload().(*CandidateBatchPayload)
	if !ok {
		t.Fatalf("payload = %T, want *CandidateBatchPayload", envelope.Payload())
	}
	if got.Source != "fusion:test" || len(got.Primary) != 1 || len(got.Candidates) != 1 {
		t.Fatalf("decoded payload = %+v", got)
	}
}

func TestFusionProjectorComponentExposesStreamAndGraphPorts(t *testing.T) {
	var _ component.LifecycleComponent = (*ProjectorComponent)(nil)

	bus := &recordingBus{}
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{Writer: writer}, bus)
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	producer := candidateProducer{name: "semops-processor-track-candidates"}

	if projector.Meta().Type != "processor" {
		t.Fatalf("projector type = %q, want processor", projector.Meta().Type)
	}
	inputPorts := projector.InputPorts()
	if len(inputPorts) != 1 {
		t.Fatalf("input ports = %d, want candidate stream", len(inputPorts))
	}
	if got, want := inputPorts[0].Config.(component.NATSPort).Subject, DefaultCandidateSubject; got != want {
		t.Fatalf("candidate subject = %q, want %q", got, want)
	}
	for _, port := range projector.OutputPorts() {
		if got, want := port.Config.Type(), "nats-request"; got != want {
			t.Fatalf("projector output port %q type = %q, want %q", port.Name, got, want)
		}
	}
	schema := projector.ConfigSchema()
	for _, key := range []string{"max_distance_meters", "max_time_delta", "min_confidence", "ambiguity_margin"} {
		if _, ok := schema.Properties[key]; !ok {
			t.Fatalf("missing fusion association config property %q: %+v", key, schema.Properties)
		}
	}
	if schema.Properties["max_distance_meters"].Default != "250" ||
		schema.Properties["max_time_delta"].Default != "10s" ||
		schema.Properties["min_confidence"].Default != "0.65" ||
		schema.Properties["ambiguity_margin"].Default != "0.05" {
		t.Fatalf("fusion association config defaults = %+v", schema.Properties)
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(producer.Meta().Name, producer); err != nil {
		t.Fatalf("add candidate producer: %v", err)
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

func TestProjectorScoresCandidateBatchAndWritesFusionPlan(t *testing.T) {
	now := time.Date(2026, 6, 23, 19, 30, 0, 0, time.UTC)
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: fusionprojector.NewProjector(fusionprojector.Config{
			OwnerTokens: map[string]ownership.OwnerToken{
				cop.OwnerFusion: ownership.ExpectedOwnerToken(cop.OwnerFusion, "component-test"),
			},
			TraceID: "fusion-component-test",
		}),
		Writer: writer,
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	if err := projector.Initialize(); err != nil {
		t.Fatalf("initialize projector: %v", err)
	}

	wire := mustBaseMessageJSON(t, CandidateBatchType, sampleCandidateBatch(now), "semops-processor-track-candidates", now)
	if err := projector.HandleCandidateMessage(context.Background(), wire); err != nil {
		t.Fatalf("handle candidate batch: %v", err)
	}

	if len(writer.plans) != 1 {
		t.Fatalf("plans = %d, want 1", len(writer.plans))
	}
	plan := writer.plans[0]
	if len(plan.Mutations) != 1 || plan.Mutations[0].Kind != fusionprojector.MutationCreate {
		t.Fatalf("plan = %+v, want fusion association create", plan)
	}
	create := plan.Mutations[0].Create
	if create.OwnerToken != "semops.fusion.structural#component-test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "fusion-component-test" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	if got := projector.DataFlow().LastActivity; got.IsZero() {
		t.Fatalf("last activity was not recorded")
	}
}

func TestProjectorReconcilesExistingFusionBirth(t *testing.T) {
	now := time.Date(2026, 6, 23, 20, 0, 0, 0, time.UTC)
	entityID := fusionassociation.EntityID(
		"c360",
		"edge",
		"c360.edge.cop.mavlink.track.system-42",
		"c360.edge.cop.adsb.track.a1b2c3",
	)
	writer := &recordingPlanWriter{
		failures: []error{
			&fusionprojector.MutationFailureError{
				Operation: "create_with_triples",
				Kind:      fusionprojector.MutationCreate,
				EntityID:  entityID,
				ErrorCode: graph.ErrorCodeEntityExists,
				Message:   "already exists",
			},
		},
	}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Projector: fusionprojector.NewProjector(fusionprojector.Config{}),
		Writer:    writer,
		Clock:     func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}

	if err := projector.HandleCandidatePayload(context.Background(), sampleCandidateBatch(now)); err != nil {
		t.Fatalf("handle candidate batch with existing birth: %v", err)
	}
	if len(writer.plans) != 2 {
		t.Fatalf("plans = %d, want create retry then update", len(writer.plans))
	}
	last := writer.plans[len(writer.plans)-1]
	if len(last.Mutations) != 1 || last.Mutations[0].Kind != fusionprojector.MutationUpdate {
		t.Fatalf("last plan = %+v, want update after reconciling existing birth", last)
	}
}

func TestProjectorRecordsNoWriteWhenCandidatesDoNotAssociate(t *testing.T) {
	now := time.Date(2026, 6, 23, 20, 30, 0, 0, time.UTC)
	writer := &recordingPlanWriter{}
	projector, err := NewProjectorComponent(ProjectorConfig{
		Writer: writer,
		Clock:  func() time.Time { return now },
	}, &recordingBus{})
	if err != nil {
		t.Fatalf("new projector: %v", err)
	}
	payload := sampleCandidateBatch(now)
	payload.Candidates[0].Source = "mavlink"
	if err := projector.HandleCandidatePayload(context.Background(), payload); err != nil {
		t.Fatalf("handle same-source candidate batch: %v", err)
	}
	if len(writer.plans) != 0 {
		t.Fatalf("plans = %d, want no graph writes for same-source candidates", len(writer.plans))
	}
	if got := projector.DataFlow().LastActivity; got.IsZero() {
		t.Fatalf("last activity was not recorded")
	}
}

func sampleCandidateBatch(observed time.Time) *CandidateBatchPayload {
	return NewCandidateBatchPayload(
		"fusion:test",
		"batch-1",
		observed,
		[]fusionassociation.TrackObservation{{
			ID:         "c360.edge.cop.mavlink.track.system-42",
			Source:     "mavlink",
			NativeID:   "system-42",
			Position:   fusionassociation.GeoPoint{Lat: 38.9001, Lon: -77.0002},
			ObservedAt: observed.Add(-2 * time.Second),
			Confidence: 1,
			SourceRef:  "mavlink://raw/udp/0001",
		}},
		[]fusionassociation.TrackObservation{{
			ID:         "c360.edge.cop.adsb.track.a1b2c3",
			Source:     "adsb",
			NativeID:   "a1b2c3",
			Position:   fusionassociation.GeoPoint{Lat: 38.90028, Lon: -77.00005},
			ObservedAt: observed,
			Confidence: 0.88,
			SourceRef:  "adsb://opensky/state/0001",
		}},
	)
}

type recordingPlanWriter struct {
	plans    []fusionprojector.Plan
	failures []error
}

func (w *recordingPlanWriter) Apply(_ context.Context, plan fusionprojector.Plan) error {
	w.plans = append(w.plans, plan)
	if len(w.failures) == 0 {
		return nil
	}
	err := w.failures[0]
	w.failures = w.failures[1:]
	return err
}

type recordingBus struct {
	handlers map[string]func(context.Context, *nats.Msg)
}

func (b *recordingBus) Publish(context.Context, string, []byte) error {
	return nil
}

func (b *recordingBus) Subscribe(
	_ context.Context,
	subject string,
	handler func(context.Context, *nats.Msg),
) (Subscription, error) {
	if b.handlers == nil {
		b.handlers = map[string]func(context.Context, *nats.Msg){}
	}
	b.handlers[subject] = handler
	return recordingSubscription{}, nil
}

type recordingSubscription struct{}

func (recordingSubscription) Unsubscribe() error {
	return nil
}

type candidateProducer struct {
	name string
}

func (p candidateProducer) Initialize() error {
	return nil
}

func (p candidateProducer) Start(context.Context) error {
	return nil
}

func (p candidateProducer) Stop(time.Duration) error {
	return nil
}

func (p candidateProducer) Meta() component.Metadata {
	return component.Metadata{Name: p.name, Type: "processor", Description: "test candidate producer", Version: "test"}
}

func (candidateProducer) InputPorts() []component.Port {
	return nil
}

func (candidateProducer) OutputPorts() []component.Port {
	return []component.Port{streamPort("track_candidates", component.DirectionOutput, DefaultCandidateSubject, CandidateBatchType)}
}

func (candidateProducer) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{}
}

func (candidateProducer) Health() component.HealthStatus {
	return component.HealthStatus{Healthy: true, Status: component.StateStarted.String()}
}

func (candidateProducer) DataFlow() component.FlowMetrics {
	return component.FlowMetrics{}
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
