package contracts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	cotcomponent "github.com/c360studio/semops/internal/components/cot"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	mavprojector "github.com/c360studio/semops/internal/projectors/mavlink"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/nats-io/nats.go"
)

func TestCurrentStateTrackProjectionUsesModernSemStreamsContracts(t *testing.T) {
	messageType := message.Type{
		Domain:   "semops",
		Category: "track",
		Version:  "v1",
	}

	contract := cop.MAVLinkTrackContract()
	contract.MessageType = messageType.Key()

	if err := contract.Validate(); err != nil {
		t.Fatalf("projection contract should validate: %v", err)
	}

	registration, err := projection.Derive(cop.OwnerMAVLink, contract)
	if err != nil {
		t.Fatalf("projection contract should derive ownership: %v", err)
	}
	if registration.Owner != cop.OwnerMAVLink {
		t.Fatalf("registration owner = %q, want %q", registration.Owner, cop.OwnerMAVLink)
	}
	if len(registration.Claims) != 1 {
		t.Fatalf("derived claims = %d, want 1", len(registration.Claims))
	}
	if len(registration.ForeignEdges) != 1 {
		t.Fatalf("derived foreign edges = %d, want 1", len(registration.ForeignEdges))
	}

	trackID := message.EntityID{
		Org:      "c360",
		Platform: "edge",
		Domain:   "cop",
		System:   "mavlink",
		Type:     "track",
		Instance: "vehicle-1",
	}.Key()
	observedAt := time.Now().UTC()
	triples := []message.Triple{{
		Subject:    trackID,
		Predicate:  cop.TrackPosition,
		Object:     "POINT(-97.7431 30.2672)",
		Source:     "mavlink",
		Timestamp:  observedAt,
		Confidence: 1.0,
	}}

	create := graph.CreateEntityWithTriplesRequest{
		Entity: &graph.EntityState{
			ID:          trackID,
			MessageType: messageType,
			UpdatedAt:   observedAt,
		},
		Triples:         triples,
		IndexingProfile: contract.IndexingProfile,
		TraceID:         "scenario-001",
		RequestID:       "create-track-vehicle-1",
	}
	if create.IndexingProfile != "signal" {
		t.Fatalf("create indexing profile = %q, want signal", create.IndexingProfile)
	}

	update := graph.UpdateEntityWithTriplesRequest{
		Entity:          &graph.EntityState{ID: trackID},
		AddTriples:      triples,
		IndexingProfile: contract.IndexingProfile,
		TraceID:         "scenario-001",
		RequestID:       "update-track-vehicle-1",
	}
	if update.AddTriples[0].Predicate != cop.TrackPosition {
		t.Fatalf("update predicate = %q, want %s", update.AddTriples[0].Predicate, cop.TrackPosition)
	}
}

func TestFeedBoundaryUsesInputAndProcessorComponentShape(t *testing.T) {
	var _ component.LifecycleComponent = (*mavcomponent.UDPInputComponent)(nil)
	var _ component.LifecycleComponent = (*mavcomponent.DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*mavcomponent.ProjectorComponent)(nil)

	bus := contractBus{}
	input, err := mavcomponent.NewUDPInputComponent(mavcomponent.UDPInputConfig{
		ListenAddr: "127.0.0.1:0",
	}, bus)
	if err != nil {
		t.Fatalf("new input component: %v", err)
	}
	decoder, err := mavcomponent.NewDecoderComponent(mavcomponent.DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new decoder component: %v", err)
	}
	projector, err := mavcomponent.NewProjectorComponent(mavcomponent.ProjectorConfig{
		Writer: contractPlanWriter{},
	}, bus)
	if err != nil {
		t.Fatalf("new projector component: %v", err)
	}

	for name, lifecycle := range map[string]component.LifecycleComponent{
		"input":     input,
		"decoder":   decoder,
		"projector": projector,
	} {
		if err := lifecycle.Initialize(); err != nil {
			t.Fatalf("initialize %s: %v", name, err)
		}
		if err := lifecycle.Start(context.Background()); err != nil {
			t.Fatalf("start %s: %v", name, err)
		}
		if err := lifecycle.Stop(time.Second); err != nil {
			t.Fatalf("stop %s: %v", name, err)
		}
	}

	if input.Meta().Type != "input" {
		t.Fatalf("input component type = %q, want input", input.Meta().Type)
	}
	if decoder.Meta().Type != "processor" {
		t.Fatalf("decoder component type = %q, want processor", decoder.Meta().Type)
	}
	if projector.Meta().Type != "processor" {
		t.Fatalf("projector component type = %q, want processor", projector.Meta().Type)
	}
	if got, want := input.InputPorts()[0].Config.Type(), "network"; got != want {
		t.Fatalf("input ingress port type = %q, want %q", got, want)
	}
	if got, want := input.OutputPorts()[0].Config.Type(), "nats"; got != want {
		t.Fatalf("input raw output port type = %q, want %q", got, want)
	}
	if got, want := decoder.InputPorts()[0].Config.Type(), "nats"; got != want {
		t.Fatalf("decoder raw input port type = %q, want %q", got, want)
	}
	if got, want := decoder.OutputPorts()[0].Config.Type(), "nats"; got != want {
		t.Fatalf("decoder decoded output port type = %q, want %q", got, want)
	}
	for _, port := range projector.OutputPorts() {
		if got, want := port.Config.Type(), "nats-request"; got != want {
			t.Fatalf("projector output port %q type = %q, want %q", port.Name, got, want)
		}
	}

	requireProperty(t, input.ConfigSchema(), "listen_addr")
	requireProperty(t, input.ConfigSchema(), "raw_subject")
	requireProperty(t, decoder.ConfigSchema(), "raw_max_records")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add input component to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add decoder component to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add projector component to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect feed flow graph: %v", err)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: input.Meta().Name,
			PortName:      "raw_frames",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: decoder.Meta().Name,
			PortName:      "raw_frames",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: mavcomponent.DefaultRawSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: decoder.Meta().Name,
			PortName:      "decoded_packets",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: projector.Meta().Name,
			PortName:      "decoded_packets",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: mavcomponent.DefaultDecodedSubject,
	})
}

func TestCoTFeedBoundaryUsesInputAndProcessorComponentShape(t *testing.T) {
	var _ component.LifecycleComponent = (*cotcomponent.UDPInputComponent)(nil)
	var _ component.LifecycleComponent = (*cotcomponent.TCPInputComponent)(nil)
	var _ component.LifecycleComponent = (*cotcomponent.DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*cotcomponent.ProjectorComponent)(nil)

	bus := cotContractBus{}
	udpInput, err := cotcomponent.NewUDPInputComponent(cotcomponent.UDPInputConfig{
		ListenAddr:        "127.0.0.1:0",
		AdvertisedUDPPort: 8087,
	}, bus)
	if err != nil {
		t.Fatalf("new CoT UDP input component: %v", err)
	}
	tcpInput, err := cotcomponent.NewTCPInputComponent(cotcomponent.TCPInputConfig{
		ListenAddr:        "127.0.0.1:0",
		AdvertisedTCPPort: 8088,
	}, bus)
	if err != nil {
		t.Fatalf("new CoT TCP input component: %v", err)
	}
	decoder, err := cotcomponent.NewDecoderComponent(cotcomponent.DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new CoT decoder component: %v", err)
	}
	projector, err := cotcomponent.NewProjectorComponent(cotcomponent.ProjectorConfig{
		Writer: cotContractPlanWriter{},
	}, bus)
	if err != nil {
		t.Fatalf("new CoT projector component: %v", err)
	}

	for name, lifecycle := range map[string]component.LifecycleComponent{
		"udp_input": udpInput,
		"tcp_input": tcpInput,
		"decoder":   decoder,
		"projector": projector,
	} {
		if err := lifecycle.Initialize(); err != nil {
			t.Fatalf("initialize %s: %v", name, err)
		}
		if err := lifecycle.Start(context.Background()); err != nil {
			t.Fatalf("start %s: %v", name, err)
		}
		if err := lifecycle.Stop(time.Second); err != nil {
			t.Fatalf("stop %s: %v", name, err)
		}
	}

	if udpInput.Meta().Type != "input" || tcpInput.Meta().Type != "input" {
		t.Fatalf("CoT input component types = %q/%q, want input/input", udpInput.Meta().Type, tcpInput.Meta().Type)
	}
	if decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf("CoT processor component types = %q/%q, want processor/processor", decoder.Meta().Type, projector.Meta().Type)
	}
	if got, want := udpInput.InputPorts()[0].Config.Type(), "network"; got != want {
		t.Fatalf("CoT UDP ingress port type = %q, want %q", got, want)
	}
	if got, want := tcpInput.InputPorts()[0].Config.Type(), "network"; got != want {
		t.Fatalf("CoT TCP ingress port type = %q, want %q", got, want)
	}
	if got, want := udpInput.OutputPorts()[0].Config.Type(), "nats"; got != want {
		t.Fatalf("CoT input raw output port type = %q, want %q", got, want)
	}
	if got, want := decoder.OutputPorts()[0].Config.Type(), "nats"; got != want {
		t.Fatalf("CoT decoder decoded output port type = %q, want %q", got, want)
	}
	for _, port := range projector.OutputPorts() {
		if got, want := port.Config.Type(), "nats-request"; got != want {
			t.Fatalf("CoT projector output port %q type = %q, want %q", port.Name, got, want)
		}
	}

	requireProperty(t, udpInput.ConfigSchema(), "listen_addr")
	requireProperty(t, tcpInput.ConfigSchema(), "max_event_bytes")
	requireProperty(t, decoder.ConfigSchema(), "raw_max_records")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(udpInput.Meta().Name, udpInput); err != nil {
		t.Fatalf("add CoT UDP input to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(tcpInput.Meta().Name, tcpInput); err != nil {
		t.Fatalf("add CoT TCP input to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add CoT decoder to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add CoT projector to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect CoT feed flow graph: %v", err)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: udpInput.Meta().Name,
			PortName:      "raw_events",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: decoder.Meta().Name,
			PortName:      "raw_events",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: cotcomponent.DefaultRawSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: tcpInput.Meta().Name,
			PortName:      "raw_events",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: decoder.Meta().Name,
			PortName:      "raw_events",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: cotcomponent.DefaultRawSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: decoder.Meta().Name,
			PortName:      "decoded_events",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: projector.Meta().Name,
			PortName:      "decoded_events",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: cotcomponent.DefaultDecodedSubject,
	})
}

func TestRawFeedFlowUsesRegisteredBaseMessagePayload(t *testing.T) {
	registry := payloadregistry.New()
	if err := mavcomponent.RegisterPayloads(registry); err != nil {
		t.Fatalf("register MAVLink payloads: %v", err)
	}

	payload := mavcomponent.NewRawFramePayload(
		"udp://0.0.0.0:14550",
		"127.0.0.1:14551",
		time.Now().UTC(),
		[]byte{0xfd, 0x00, 0x00},
	)
	wire, err := message.NewBaseMessage(
		mavcomponent.RawFrameType,
		payload,
		"semops-input-mavlink-udp",
	).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal raw feed BaseMessage: %v", err)
	}

	decoded, err := message.NewDecoder(registry).Decode(wire)
	if err != nil {
		t.Fatalf("decode raw feed BaseMessage: %v", err)
	}
	got, ok := decoded.Payload().(*mavcomponent.RawFramePayload)
	if !ok {
		t.Fatalf("decoded payload type = %T, want *RawFramePayload", decoded.Payload())
	}
	if got.Source != payload.Source || string(got.Frame) != string(payload.Frame) {
		t.Fatalf("decoded payload = %+v, want %+v", got, payload)
	}
}

func TestCoTRawFeedFlowUsesRegisteredBaseMessagePayload(t *testing.T) {
	registry := payloadregistry.New()
	if err := cotcomponent.RegisterPayloads(registry); err != nil {
		t.Fatalf("register CoT payloads: %v", err)
	}

	now := time.Now().UTC()
	raw, err := cotcodec.Marshal(cotcodec.Event{
		UID:      "ANDROID-ALPHA",
		Type:     cotcodec.TypeOperatorPosition,
		Time:     now,
		Stale:    now.Add(2 * time.Minute),
		Callsign: "Alpha Team",
		Point:    &cotcodec.Point{Lat: 30.2672, Lon: -97.7431},
	})
	if err != nil {
		t.Fatalf("marshal CoT event: %v", err)
	}
	payload := cotcomponent.NewRawEventPayload(
		"udp://0.0.0.0:8087",
		"127.0.0.1:50000",
		now,
		raw,
	)
	wire, err := message.NewBaseMessage(
		cotcomponent.RawEventType,
		payload,
		"semops-input-cot-udp",
	).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal CoT raw feed BaseMessage: %v", err)
	}

	decoded, err := message.NewDecoder(registry).Decode(wire)
	if err != nil {
		t.Fatalf("decode CoT raw feed BaseMessage: %v", err)
	}
	got, ok := decoded.Payload().(*cotcomponent.RawEventPayload)
	if !ok {
		t.Fatalf("decoded payload type = %T, want *RawEventPayload", decoded.Payload())
	}
	if got.Source != payload.Source || string(got.RawXML) != string(payload.RawXML) {
		t.Fatalf("decoded payload = %+v, want %+v", got, payload)
	}
}

func TestLegacyRoboticsFlowConfigIsNotRetained(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "robotics-flow.json")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s must not be retained; use SemStreams component metadata, flowgraph, payload registry, ports, and config schema instead", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func requireProperty(t *testing.T, schema component.ConfigSchema, property string) {
	t.Helper()
	if _, ok := schema.Properties[property]; !ok {
		t.Fatalf("config schema missing %q: %+v", property, schema.Properties)
	}
}

func requireFlowEdge(t *testing.T, edges []flowgraph.FlowEdge, want flowgraph.FlowEdge) {
	t.Helper()
	for _, edge := range edges {
		if edge.From == want.From &&
			edge.To == want.To &&
			edge.Pattern == want.Pattern &&
			edge.ConnectionID == want.ConnectionID {
			return
		}
	}
	t.Fatalf("missing flow edge %+v in %+v", want, edges)
}

type contractBus struct{}

func (contractBus) Publish(context.Context, string, []byte) error {
	return nil
}

func (contractBus) Subscribe(
	context.Context,
	string,
	func(context.Context, *nats.Msg),
) (mavcomponent.Subscription, error) {
	return contractSubscription{}, nil
}

type contractSubscription struct{}

func (contractSubscription) Unsubscribe() error {
	return nil
}

type contractPlanWriter struct{}

func (contractPlanWriter) Apply(context.Context, mavprojector.Plan) error {
	return nil
}

type cotContractBus struct{}

func (cotContractBus) Publish(context.Context, string, []byte) error {
	return nil
}

func (cotContractBus) Subscribe(
	context.Context,
	string,
	func(context.Context, *nats.Msg),
) (cotcomponent.Subscription, error) {
	return contractSubscription{}, nil
}

type cotContractPlanWriter struct{}

func (cotContractPlanWriter) Apply(context.Context, cotprojector.Plan) error {
	return nil
}
