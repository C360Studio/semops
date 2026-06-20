package contracts

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/component/flowgraph"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/projection"
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
	var _ component.LifecycleComponent = (*transportInputSkeleton)(nil)
	var _ component.LifecycleComponent = (*feedProcessorSkeleton)(nil)
	var _ component.LifecycleComponent = (*projectionProcessorSkeleton)(nil)

	input := &transportInputSkeleton{}
	if err := input.Initialize(); err != nil {
		t.Fatalf("initialize input should be a no-op in the skeleton: %v", err)
	}
	if err := input.Start(context.Background()); err != nil {
		t.Fatalf("start input should accept caller context: %v", err)
	}
	if err := input.Stop(time.Second); err != nil {
		t.Fatalf("stop input should accept a timeout: %v", err)
	}

	inputMeta := input.Meta()
	if inputMeta.Type != "input" {
		t.Fatalf("input component type = %q, want input", inputMeta.Type)
	}
	inputPorts := input.InputPorts()
	if len(inputPorts) != 1 {
		t.Fatalf("input network ports = %d, want 1", len(inputPorts))
	}
	if got, want := inputPorts[0].Config.Type(), "network"; got != want {
		t.Fatalf("input port type = %q, want %q", got, want)
	}
	rawOutputs := input.OutputPorts()
	if len(rawOutputs) != 1 {
		t.Fatalf("raw output ports = %d, want 1", len(rawOutputs))
	}
	if got, want := rawOutputs[0].Config.Type(), "nats"; got != want {
		t.Fatalf("raw output port type = %q, want %q", got, want)
	}

	processor := &feedProcessorSkeleton{}
	if err := processor.Initialize(); err != nil {
		t.Fatalf("initialize processor should be a no-op in the skeleton: %v", err)
	}
	if err := processor.Start(context.Background()); err != nil {
		t.Fatalf("start processor should accept caller context: %v", err)
	}
	if err := processor.Stop(time.Second); err != nil {
		t.Fatalf("stop processor should accept a timeout: %v", err)
	}

	processorMeta := processor.Meta()
	if processorMeta.Type != "processor" {
		t.Fatalf("processor component type = %q, want processor", processorMeta.Type)
	}

	inputs := processor.InputPorts()
	if len(inputs) != 1 {
		t.Fatalf("processor input ports = %d, want 1", len(inputs))
	}
	if got, want := inputs[0].Config.Type(), "nats"; got != want {
		t.Fatalf("processor input port type = %q, want %q", got, want)
	}

	outputs := processor.OutputPorts()
	if len(outputs) != 1 {
		t.Fatalf("processor output ports = %d, want 1", len(outputs))
	}
	if got, want := outputs[0].Config.Type(), "nats"; got != want {
		t.Fatalf("decoded output port type = %q, want %q", got, want)
	}

	inputSchema := input.ConfigSchema()
	if _, ok := inputSchema.Properties["listen_addr"]; !ok {
		t.Fatalf("input config schema missing listen_addr: %+v", inputSchema.Properties)
	}
	if _, ok := inputSchema.Properties["raw_subject"]; !ok {
		t.Fatalf("input config schema missing raw_subject: %+v", inputSchema.Properties)
	}

	schema := processor.ConfigSchema()
	if _, ok := schema.Properties["raw_max_records"]; !ok {
		t.Fatalf("processor config schema missing raw_max_records: %+v", schema.Properties)
	}
	if _, ok := schema.Properties["decoded_subject"]; !ok {
		t.Fatalf("processor config schema missing decoded_subject: %+v", schema.Properties)
	}

	projector := &projectionProcessorSkeleton{}
	if err := projector.Initialize(); err != nil {
		t.Fatalf("initialize projector should be a no-op in the skeleton: %v", err)
	}
	if err := projector.Start(context.Background()); err != nil {
		t.Fatalf("start projector should accept caller context: %v", err)
	}
	if err := projector.Stop(time.Second); err != nil {
		t.Fatalf("stop projector should accept a timeout: %v", err)
	}
	projectorMeta := projector.Meta()
	if projectorMeta.Type != "processor" {
		t.Fatalf("projector component type = %q, want processor", projectorMeta.Type)
	}
	projectorInputs := projector.InputPorts()
	if len(projectorInputs) != 1 {
		t.Fatalf("projector input ports = %d, want 1", len(projectorInputs))
	}
	if got, want := projectorInputs[0].Config.Type(), "nats"; got != want {
		t.Fatalf("projector input port type = %q, want %q", got, want)
	}
	projectorOutputs := projector.OutputPorts()
	if len(projectorOutputs) != 2 {
		t.Fatalf("projector output ports = %d, want 2", len(projectorOutputs))
	}
	for _, port := range projectorOutputs {
		if got, want := port.Config.Type(), "nats-request"; got != want {
			t.Fatalf("projector output port %q type = %q, want %q", port.Name, got, want)
		}
	}
	projectorSchema := projector.ConfigSchema()
	if _, ok := projectorSchema.Properties["owner"]; !ok {
		t.Fatalf("projector config schema missing owner: %+v", projectorSchema.Properties)
	}

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode("semops-input-mavlink-udp", input); err != nil {
		t.Fatalf("add input component to flow graph: %v", err)
	}
	if err := fg.AddComponentNode("semops-processor-mavlink", processor); err != nil {
		t.Fatalf("add processor component to flow graph: %v", err)
	}
	if err := fg.AddComponentNode("semops-projector-mavlink", projector); err != nil {
		t.Fatalf("add projector component to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect feed flow graph: %v", err)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: "semops-input-mavlink-udp",
			PortName:      "raw_frames",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: "semops-processor-mavlink",
			PortName:      "raw_frames",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: "semops.feed.mavlink.raw",
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From: flowgraph.ComponentPortRef{
			ComponentName: "semops-processor-mavlink",
			PortName:      "decoded_packets",
		},
		To: flowgraph.ComponentPortRef{
			ComponentName: "semops-projector-mavlink",
			PortName:      "decoded_packets",
		},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: "semops.feed.mavlink.decoded",
	})
}

func TestRawFeedFlowUsesRegisteredBaseMessagePayload(t *testing.T) {
	registry := payloadregistry.New()
	if err := registry.Register(&payloadregistry.Registration{
		Factory:     func() any { return &rawMAVLinkFramePayload{} },
		Domain:      "semops",
		Category:    "mavlink_raw_frame",
		Version:     "v1",
		Description: "Raw MAVLink frame captured by a SemOps input component",
	}); err != nil {
		t.Fatalf("register raw MAVLink payload: %v", err)
	}

	payload := &rawMAVLinkFramePayload{
		Source: "udp://0.0.0.0:14550",
		Frame:  []byte{0xfd, 0x00, 0x00},
	}
	envelope := message.NewBaseMessage(payload.Schema(), payload, "semops-input-mavlink-udp")
	wire, err := envelope.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal raw feed BaseMessage: %v", err)
	}

	decoded, err := message.NewDecoder(registry).Decode(wire)
	if err != nil {
		t.Fatalf("decode raw feed BaseMessage: %v", err)
	}
	got, ok := decoded.Payload().(*rawMAVLinkFramePayload)
	if !ok {
		t.Fatalf("decoded payload type = %T, want *rawMAVLinkFramePayload", decoded.Payload())
	}
	if got.Source != payload.Source || string(got.Frame) != string(payload.Frame) {
		t.Fatalf("decoded payload = %+v, want %+v", got, payload)
	}
}

func TestLegacyRoboticsFlowConfigIsNotRetained(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "robotics-flow.json")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%s must not be retained; use SemStreams component metadata, ports, and config schema instead", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
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

type rawMAVLinkFramePayload struct {
	Source string `json:"source"`
	Frame  []byte `json:"frame"`
}

func (p *rawMAVLinkFramePayload) Schema() message.Type {
	return message.Type{Domain: "semops", Category: "mavlink_raw_frame", Version: "v1"}
}

func (p *rawMAVLinkFramePayload) Validate() error {
	if p == nil {
		return errors.New("payload is nil")
	}
	if p.Source == "" {
		return errors.New("source is required")
	}
	if len(p.Frame) == 0 {
		return errors.New("frame is required")
	}
	return nil
}

func (p *rawMAVLinkFramePayload) MarshalJSON() ([]byte, error) {
	type alias rawMAVLinkFramePayload
	return json.Marshal((*alias)(p))
}

func (p *rawMAVLinkFramePayload) UnmarshalJSON(data []byte) error {
	type alias rawMAVLinkFramePayload
	return json.Unmarshal(data, (*alias)(p))
}

type transportInputSkeleton struct{}

func (transportInputSkeleton) Meta() component.Metadata {
	return component.Metadata{
		Name:        "semops-input-mavlink-udp",
		Type:        "input",
		Description: "MAVLink UDP input component for SemOps COP feeds",
		Version:     "v0",
	}
}

func (transportInputSkeleton) InputPorts() []component.Port {
	return []component.Port{{
		Name:        "mavlink_datagrams",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "MAVLink UDP datagram ingress owned by the SemOps input component",
		Config: component.NetworkPort{
			Protocol: "udp",
			Host:     "0.0.0.0",
			Port:     14550,
		},
	}}
}

func (transportInputSkeleton) OutputPorts() []component.Port {
	return []component.Port{{
		Name:        "raw_frames",
		Direction:   component.DirectionOutput,
		Required:    true,
		Description: "Raw MAVLink frames handed to processor components through a declared SemStreams port",
		Config: component.NATSPort{
			Subject: "semops.feed.mavlink.raw",
			Interface: &component.InterfaceContract{
				Type:    "message.BaseMessage",
				Version: "v1",
				Compatible: []string{
					"semops.mavlink_raw_frame.v1",
				},
			},
		},
	}}
}

func (transportInputSkeleton) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"listen_addr": {
				Type:        "string",
				Description: "UDP address for the MAVLink transport input component",
				Default:     ":14550",
			},
			"raw_subject": {
				Type:        "string",
				Description: "SemStreams subject carrying raw MAVLink frames to processors",
				Default:     "semops.feed.mavlink.raw",
			},
			"max_datagram_bytes": {
				Type:        "int",
				Description: "Maximum accepted UDP datagram size",
				Default:     2048,
			},
		},
		Required: []string{"listen_addr", "raw_subject"},
	}
}

func (transportInputSkeleton) Health() component.HealthStatus {
	return component.HealthStatus{Healthy: true, Status: "not-started"}
}

func (transportInputSkeleton) DataFlow() component.FlowMetrics {
	return component.FlowMetrics{}
}

func (transportInputSkeleton) Initialize() error {
	return nil
}

func (transportInputSkeleton) Start(context.Context) error {
	return nil
}

func (transportInputSkeleton) Stop(time.Duration) error {
	return nil
}

type feedProcessorSkeleton struct{}

func (feedProcessorSkeleton) Meta() component.Metadata {
	return component.Metadata{
		Name:        "semops-processor-mavlink",
		Type:        "processor",
		Description: "MAVLink parser/decoder processor for SemOps COP feeds",
		Version:     "v0",
	}
}

func (feedProcessorSkeleton) InputPorts() []component.Port {
	return []component.Port{{
		Name:        "raw_frames",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Declared raw MAVLink subject consumed from transport input components",
		Config: component.NATSPort{
			Subject: "semops.feed.mavlink.raw",
			Queue:   "semops-mavlink-processors",
			Interface: &component.InterfaceContract{
				Type:    "message.BaseMessage",
				Version: "v1",
				Compatible: []string{
					"semops.mavlink_raw_frame.v1",
				},
			},
		},
	}}
}

func (feedProcessorSkeleton) OutputPorts() []component.Port {
	return []component.Port{{
		Name:        "decoded_packets",
		Direction:   component.DirectionOutput,
		Required:    true,
		Description: "Decoded MAVLink packets emitted as a tappable SemStreams stream",
		Config: component.NATSPort{
			Subject: "semops.feed.mavlink.decoded",
			Interface: &component.InterfaceContract{
				Type:    "message.BaseMessage",
				Version: "v1",
				Compatible: []string{
					"semops.mavlink_packet.v1",
				},
			},
		},
	}}
}

func (feedProcessorSkeleton) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"raw_max_records": {
				Type:        "int",
				Description: "Maximum retained raw records before eviction",
				Default:     1024,
			},
			"raw_max_bytes": {
				Type:        "int",
				Description: "Maximum retained raw bytes before eviction",
				Default:     8 * 1024 * 1024,
			},
			"decoded_subject": {
				Type:        "string",
				Description: "SemStreams subject carrying decoded MAVLink packets to downstream components",
				Default:     "semops.feed.mavlink.decoded",
			},
		},
		Required: []string{"decoded_subject"},
	}
}

func (feedProcessorSkeleton) Health() component.HealthStatus {
	return component.HealthStatus{Healthy: true, Status: "not-started"}
}

func (feedProcessorSkeleton) DataFlow() component.FlowMetrics {
	return component.FlowMetrics{}
}

func (feedProcessorSkeleton) Initialize() error {
	return nil
}

func (feedProcessorSkeleton) Start(context.Context) error {
	return nil
}

func (feedProcessorSkeleton) Stop(time.Duration) error {
	return nil
}

type projectionProcessorSkeleton struct{}

func (projectionProcessorSkeleton) Meta() component.Metadata {
	return component.Metadata{
		Name:        "semops-projector-mavlink",
		Type:        "processor",
		Description: "MAVLink governed graph projection processor for SemOps COP feeds",
		Version:     "v0",
	}
}

func (projectionProcessorSkeleton) InputPorts() []component.Port {
	return []component.Port{{
		Name:        "decoded_packets",
		Direction:   component.DirectionInput,
		Required:    true,
		Description: "Declared decoded MAVLink stream consumed from parser components",
		Config: component.NATSPort{
			Subject: "semops.feed.mavlink.decoded",
			Queue:   "semops-mavlink-projectors",
			Interface: &component.InterfaceContract{
				Type:    "message.BaseMessage",
				Version: "v1",
				Compatible: []string{
					"semops.mavlink_packet.v1",
				},
			},
		},
	}}
}

func (projectionProcessorSkeleton) OutputPorts() []component.Port {
	return []component.Port{
		{
			Name:        "graph_create",
			Direction:   component.DirectionOutput,
			Required:    true,
			Description: "SemStreams born-first graph mutation request",
			Config: component.NATSRequestPort{
				Subject: "graph.mutation.entity.create_with_triples",
				Timeout: "5s",
				Retries: 3,
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
				Subject: "graph.mutation.entity.update_with_triples",
				Timeout: "5s",
				Retries: 3,
				Interface: &component.InterfaceContract{
					Type:    "graph.UpdateEntityWithTriplesRequest",
					Version: "v1",
				},
			},
		},
	}
}

func (projectionProcessorSkeleton) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{
		Properties: map[string]component.PropertySchema{
			"owner": {
				Type:        "string",
				Description: "SemStreams projection owner bound through registry/heartbeat before graph writes",
				Default:     cop.OwnerMAVLink,
			},
		},
		Required: []string{"owner"},
	}
}

func (projectionProcessorSkeleton) Health() component.HealthStatus {
	return component.HealthStatus{Healthy: true, Status: "not-started"}
}

func (projectionProcessorSkeleton) DataFlow() component.FlowMetrics {
	return component.FlowMetrics{}
}

func (projectionProcessorSkeleton) Initialize() error {
	return nil
}

func (projectionProcessorSkeleton) Start(context.Context) error {
	return nil
}

func (projectionProcessorSkeleton) Stop(time.Duration) error {
	return nil
}
