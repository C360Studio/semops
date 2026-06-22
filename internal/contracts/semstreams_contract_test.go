package contracts

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	adsbcomponent "github.com/c360studio/semops/internal/components/adsb"
	capcomponent "github.com/c360studio/semops/internal/components/cap"
	cotcomponent "github.com/c360studio/semops/internal/components/cot"
	klvcomponent "github.com/c360studio/semops/internal/components/klv"
	mavcomponent "github.com/c360studio/semops/internal/components/mavlink"
	sapientcomponent "github.com/c360studio/semops/internal/components/sapient"
	adsbprojector "github.com/c360studio/semops/internal/projectors/adsb"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
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
		t.Fatalf(
			"CoT processor component types = %q/%q, want processor/processor",
			decoder.Meta().Type,
			projector.Meta().Type,
		)
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

func TestExternalHTTPPollingBoundaryUsesSemStreamsHTTPClientPort(t *testing.T) {
	var _ component.Portable = component.HTTPClientPort{}
	var _ component.LifecycleComponent = (*capcomponent.HTTPPollerComponent)(nil)
	var _ component.LifecycleComponent = (*capcomponent.DecoderComponent)(nil)

	bus := capContractBus{}
	poller, err := capcomponent.NewHTTPPollerComponent(capcomponent.HTTPPollerConfig{
		URL:           "https://api.weather.gov/alerts/active",
		PollInterval:  30 * time.Second,
		AuthRef:       "nws-alerts",
		ContactPolicy: "semops-demo@example.invalid",
	}, bus)
	if err != nil {
		t.Fatalf("new CAP HTTP poller: %v", err)
	}
	decoder, err := capcomponent.NewDecoderComponent(capcomponent.DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new CAP decoder: %v", err)
	}
	projector, err := capcomponent.NewProjectorComponent(capcomponent.ProjectorConfig{
		Writer: capContractPlanWriter{},
	}, bus)
	if err != nil {
		t.Fatalf("new CAP projector: %v", err)
	}
	for name, lifecycle := range map[string]component.LifecycleComponent{
		"poller":    poller,
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

	if poller.Meta().Type != "input" {
		t.Fatalf("CAP poller component type = %q, want input", poller.Meta().Type)
	}
	if decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf(
			"CAP processor component types = %q/%q, want processor/processor",
			decoder.Meta().Type,
			projector.Meta().Type,
		)
	}
	capFeed, ok := poller.InputPorts()[0].Config.(component.HTTPClientPort)
	if !ok {
		t.Fatalf("CAP poller cap_feed config = %T, want HTTPClientPort", poller.InputPorts()[0].Config)
	}
	if got, want := capFeed.Type(), "http-client"; got != want {
		t.Fatalf("HTTP client port type = %q, want %q", got, want)
	}
	if got, want := capFeed.ResourceID(), "http-client:GET:https://api.weather.gov/alerts/active"; got != want {
		t.Fatalf("HTTP client resource id = %q, want %q", got, want)
	}
	if capFeed.IsExclusive() {
		t.Fatalf("HTTP client port must be shareable so multiple components can poll the same external resource")
	}
	if got, want := capFeed.Interface.Compatible[0], capcomponent.RawAlertType.Key(); got != want {
		t.Fatalf("HTTP client compatible payload = %q, want %q", got, want)
	}

	tick, ok := poller.InputPorts()[1].Config.(component.TimerPort)
	if !ok {
		t.Fatalf("CAP poller poll_tick config = %T, want TimerPort", poller.InputPorts()[1].Config)
	}
	if got, want := tick.Type(), "timer"; got != want {
		t.Fatalf("timer port type = %q, want %q", got, want)
	}

	requireProperty(t, poller.ConfigSchema(), "url")
	requireProperty(t, poller.ConfigSchema(), "contact_policy")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(poller.Meta().Name, poller); err != nil {
		t.Fatalf("add HTTP poller to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add decoder to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add projector to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect HTTP poller flow graph: %v", err)
	}

	pollerNode := fg.GetNodes()[poller.Meta().Name]
	if len(pollerNode.InputPorts) != 2 {
		t.Fatalf("poller input ports = %d, want HTTP client and timer", len(pollerNode.InputPorts))
	}
	if got, want := pollerNode.InputPorts[0].Pattern, flowgraph.PatternHTTPClient; got != want {
		t.Fatalf("HTTP polling input pattern = %q, want %q", got, want)
	}
	if got, want := pollerNode.InputPorts[0].ConnectionID, capFeed.URLPattern; got != want {
		t.Fatalf("HTTP polling connection id = %q, want %q", got, want)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: poller.Meta().Name, PortName: "raw_alerts"},
		To:           flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "raw_alerts"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: capcomponent.DefaultRawSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "decoded_alerts"},
		To:           flowgraph.ComponentPortRef{ComponentName: projector.Meta().Name, PortName: "decoded_alerts"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: capcomponent.DefaultDecodedSubject,
	})

	analysis := fg.AnalyzeConnectivity()
	for _, orphan := range analysis.OrphanedPorts {
		if orphan.ComponentName == poller.Meta().Name && orphan.PortName == "cap_feed" {
			t.Fatalf("HTTP client input reported as orphaned: %+v", orphan)
		}
	}
}

func TestADSBHTTPPollingBoundaryUsesSemStreamsComponentShape(t *testing.T) {
	var _ component.Portable = component.HTTPClientPort{}
	var _ component.LifecycleComponent = (*adsbcomponent.HTTPPollerComponent)(nil)
	var _ component.LifecycleComponent = (*adsbcomponent.DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*adsbcomponent.ProjectorComponent)(nil)

	bus := adsbContractBus{}
	poller, err := adsbcomponent.NewHTTPPollerComponent(adsbcomponent.HTTPPollerConfig{
		URL:           "https://opensky-network.org/api/states/all",
		PollInterval:  30 * time.Second,
		AuthRef:       "opensky",
		ContactPolicy: "semops-demo@example.invalid",
	}, bus)
	if err != nil {
		t.Fatalf("new ADS-B HTTP poller: %v", err)
	}
	decoder, err := adsbcomponent.NewDecoderComponent(adsbcomponent.DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new ADS-B decoder: %v", err)
	}
	projector, err := adsbcomponent.NewProjectorComponent(adsbcomponent.ProjectorConfig{
		Writer: adsbContractPlanWriter{},
	}, bus)
	if err != nil {
		t.Fatalf("new ADS-B projector: %v", err)
	}
	for name, lifecycle := range map[string]component.LifecycleComponent{
		"poller":    poller,
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

	if poller.Meta().Type != "input" {
		t.Fatalf("ADS-B poller component type = %q, want input", poller.Meta().Type)
	}
	if decoder.Meta().Type != "processor" || projector.Meta().Type != "processor" {
		t.Fatalf(
			"ADS-B processor component types = %q/%q, want processor/processor",
			decoder.Meta().Type,
			projector.Meta().Type,
		)
	}
	feed, ok := poller.InputPorts()[0].Config.(component.HTTPClientPort)
	if !ok {
		t.Fatalf("ADS-B poller adsb_feed config = %T, want HTTPClientPort", poller.InputPorts()[0].Config)
	}
	if got, want := feed.Type(), "http-client"; got != want {
		t.Fatalf("HTTP client port type = %q, want %q", got, want)
	}
	if got, want := feed.ResourceID(), "http-client:GET:https://opensky-network.org/api/states/all"; got != want {
		t.Fatalf("HTTP client resource id = %q, want %q", got, want)
	}
	if feed.IsExclusive() {
		t.Fatalf("HTTP client port must be shareable so multiple components can poll the same external resource")
	}
	if got, want := feed.Interface.Compatible[0], adsbcomponent.RawSnapshotType.Key(); got != want {
		t.Fatalf("HTTP client compatible payload = %q, want %q", got, want)
	}

	tick, ok := poller.InputPorts()[1].Config.(component.TimerPort)
	if !ok {
		t.Fatalf("ADS-B poller poll_tick config = %T, want TimerPort", poller.InputPorts()[1].Config)
	}
	if got, want := tick.Type(), "timer"; got != want {
		t.Fatalf("timer port type = %q, want %q", got, want)
	}

	requireProperty(t, poller.ConfigSchema(), "url")
	requireProperty(t, poller.ConfigSchema(), "contact_policy")
	requireProperty(t, decoder.ConfigSchema(), "raw_max_records")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(poller.Meta().Name, poller); err != nil {
		t.Fatalf("add ADS-B HTTP poller to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add ADS-B decoder to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(projector.Meta().Name, projector); err != nil {
		t.Fatalf("add ADS-B projector to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect ADS-B HTTP poller flow graph: %v", err)
	}

	pollerNode := fg.GetNodes()[poller.Meta().Name]
	if len(pollerNode.InputPorts) != 2 {
		t.Fatalf("poller input ports = %d, want HTTP client and timer", len(pollerNode.InputPorts))
	}
	if got, want := pollerNode.InputPorts[0].Pattern, flowgraph.PatternHTTPClient; got != want {
		t.Fatalf("HTTP polling input pattern = %q, want %q", got, want)
	}
	if got, want := pollerNode.InputPorts[0].ConnectionID, feed.URLPattern; got != want {
		t.Fatalf("HTTP polling connection id = %q, want %q", got, want)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: poller.Meta().Name, PortName: "raw_snapshots"},
		To:           flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "raw_snapshots"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: adsbcomponent.DefaultRawSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "decoded_snapshots"},
		To:           flowgraph.ComponentPortRef{ComponentName: projector.Meta().Name, PortName: "decoded_snapshots"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: adsbcomponent.DefaultDecodedSubject,
	})

	analysis := fg.AnalyzeConnectivity()
	for _, orphan := range analysis.OrphanedPorts {
		if orphan.ComponentName == poller.Meta().Name && orphan.PortName == "adsb_feed" {
			t.Fatalf("HTTP client input reported as orphaned: %+v", orphan)
		}
	}
}

func TestSAPIENTPreflightBoundaryUsesSemStreamsComponentShapeWithoutGraphWrites(t *testing.T) {
	var _ component.Portable = component.HTTPClientPort{}
	var _ component.LifecycleComponent = (*sapientcomponent.HTTPInputComponent)(nil)
	var _ component.LifecycleComponent = (*sapientcomponent.DecoderComponent)(nil)

	bus := sapientContractBus{}
	input, err := sapientcomponent.NewHTTPInputComponent(sapientcomponent.HTTPInputConfig{
		URL:           "https://apex.example.invalid/sapient/messages",
		PollInterval:  30 * time.Second,
		AuthRef:       "sapient-apex",
		ContactPolicy: "semops-demo@example.invalid",
	}, bus)
	if err != nil {
		t.Fatalf("new SAPIENT HTTP input: %v", err)
	}
	decoder, err := sapientcomponent.NewDecoderComponent(sapientcomponent.DecoderConfig{}, bus)
	if err != nil {
		t.Fatalf("new SAPIENT decoder: %v", err)
	}
	for name, lifecycle := range map[string]component.LifecycleComponent{
		"input":   input,
		"decoder": decoder,
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
		t.Fatalf("SAPIENT input component type = %q, want input", input.Meta().Type)
	}
	if decoder.Meta().Type != "processor" {
		t.Fatalf("SAPIENT decoder component type = %q, want processor", decoder.Meta().Type)
	}
	feed, ok := input.InputPorts()[0].Config.(component.HTTPClientPort)
	if !ok {
		t.Fatalf("SAPIENT input sapient_feed config = %T, want HTTPClientPort", input.InputPorts()[0].Config)
	}
	if got, want := feed.Type(), "http-client"; got != want {
		t.Fatalf("HTTP client port type = %q, want %q", got, want)
	}
	if got, want := feed.Interface.Compatible[0], sapientcomponent.RawMessageType.Key(); got != want {
		t.Fatalf("HTTP client compatible payload = %q, want %q", got, want)
	}
	if got, want := input.InputPorts()[1].Config.Type(), "timer"; got != want {
		t.Fatalf("timer port type = %q, want %q", got, want)
	}
	if got, want := input.OutputPorts()[0].Config.(component.NATSPort).Subject,
		sapientcomponent.DefaultRawSubject; got != want {
		t.Fatalf("input raw subject = %q, want %q", got, want)
	}
	if got, want := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject,
		sapientcomponent.DefaultDecodedSubject; got != want {
		t.Fatalf("decoder decoded subject = %q, want %q", got, want)
	}
	for _, port := range decoder.OutputPorts() {
		if got, want := port.Config.Type(), "nats"; got != want {
			t.Fatalf("SAPIENT decoder output port %q type = %q, want %q", port.Name, got, want)
		}
	}

	requireProperty(t, input.ConfigSchema(), "url")
	requireProperty(t, input.ConfigSchema(), "encoding")
	requireProperty(t, decoder.ConfigSchema(), "raw_max_records")
	requireProperty(t, decoder.ConfigSchema(), "decoded_subject")

	fg := flowgraph.NewFlowGraph()
	if err := fg.AddComponentNode(input.Meta().Name, input); err != nil {
		t.Fatalf("add SAPIENT HTTP input to flow graph: %v", err)
	}
	if err := fg.AddComponentNode(decoder.Meta().Name, decoder); err != nil {
		t.Fatalf("add SAPIENT decoder to flow graph: %v", err)
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect SAPIENT HTTP input flow graph: %v", err)
	}
	inputNode := fg.GetNodes()[input.Meta().Name]
	if got, want := inputNode.InputPorts[0].Pattern, flowgraph.PatternHTTPClient; got != want {
		t.Fatalf("HTTP input pattern = %q, want %q", got, want)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: input.Meta().Name, PortName: "raw_messages"},
		To:           flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "raw_messages"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: sapientcomponent.DefaultRawSubject,
	})

	analysis := fg.AnalyzeConnectivity()
	for _, orphan := range analysis.OrphanedPorts {
		if orphan.ComponentName == input.Meta().Name && orphan.PortName == "sapient_feed" {
			t.Fatalf("HTTP client input reported as orphaned: %+v", orphan)
		}
	}
}

func TestKLVWorkerBoundaryUsesMediaRefDemuxDecodeAndProjectorComponents(t *testing.T) {
	var _ component.LifecycleComponent = (*klvcomponent.MediaRefInputComponent)(nil)
	var _ component.LifecycleComponent = (*klvcomponent.DemuxComponent)(nil)
	var _ component.LifecycleComponent = (*klvcomponent.DecoderComponent)(nil)
	var _ component.LifecycleComponent = (*klvcomponent.ProjectorComponent)(nil)

	input, err := klvcomponent.NewMediaRefInputComponent(klvcomponent.MediaRefInputConfig{})
	if err != nil {
		t.Fatalf("new KLV media-ref input: %v", err)
	}
	demux, err := klvcomponent.NewDemuxComponent(klvcomponent.DemuxConfig{})
	if err != nil {
		t.Fatalf("new KLV demux: %v", err)
	}
	decoder, err := klvcomponent.NewDecoderComponent(klvcomponent.DecoderConfig{})
	if err != nil {
		t.Fatalf("new KLV decoder: %v", err)
	}
	projector, err := klvcomponent.NewProjectorComponent(klvcomponent.ProjectorConfig{})
	if err != nil {
		t.Fatalf("new KLV projector: %v", err)
	}

	if got, want := input.InputPorts()[0].Config.Type(), "file"; got != want {
		t.Fatalf("KLV media input port type = %q, want %q", got, want)
	}
	if got, want := input.OutputPorts()[0].Config.(component.NATSPort).Subject,
		klvcomponent.DefaultMediaRefSubject; got != want {
		t.Fatalf("KLV media-ref subject = %q, want %q", got, want)
	}
	if got, want := demux.OutputPorts()[0].Config.(component.NATSPort).Subject,
		klvcomponent.DefaultPacketSubject; got != want {
		t.Fatalf("KLV packet subject = %q, want %q", got, want)
	}
	if got, want := decoder.OutputPorts()[0].Config.(component.NATSPort).Subject,
		klvcomponent.DefaultFrameSubject; got != want {
		t.Fatalf("KLV frame subject = %q, want %q", got, want)
	}
	for _, port := range projector.OutputPorts() {
		if got, want := port.Config.Type(), "nats-request"; got != want {
			t.Fatalf("KLV projector output port %q type = %q, want %q", port.Name, got, want)
		}
	}

	requireProperty(t, input.ConfigSchema(), "media_ref_subject")
	requireProperty(t, demux.ConfigSchema(), "max_packet_bytes")
	requireProperty(t, decoder.ConfigSchema(), "supported_subset")
	requireProperty(t, projector.ConfigSchema(), "owner")

	fg := flowgraph.NewFlowGraph()
	for _, comp := range []component.Discoverable{input, demux, decoder, projector} {
		if err := fg.AddComponentNode(comp.Meta().Name, comp); err != nil {
			t.Fatalf("add KLV component %s to flow graph: %v", comp.Meta().Name, err)
		}
	}
	if err := fg.ConnectComponentsByPatterns(); err != nil {
		t.Fatalf("connect KLV flow graph: %v", err)
	}
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: input.Meta().Name, PortName: "media_refs"},
		To:           flowgraph.ComponentPortRef{ComponentName: demux.Meta().Name, PortName: "media_refs"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: klvcomponent.DefaultMediaRefSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: demux.Meta().Name, PortName: "klv_packets"},
		To:           flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "klv_packets"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: klvcomponent.DefaultPacketSubject,
	})
	requireFlowEdge(t, fg.GetEdges(), flowgraph.FlowEdge{
		From:         flowgraph.ComponentPortRef{ComponentName: decoder.Meta().Name, PortName: "misb0601_frames"},
		To:           flowgraph.ComponentPortRef{ComponentName: projector.Meta().Name, PortName: "misb0601_frames"},
		Pattern:      flowgraph.PatternStream,
		ConnectionID: klvcomponent.DefaultFrameSubject,
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
		t.Fatalf(
			"%s must not be retained; use SemStreams component metadata, flowgraph, "+
				"payload registry, ports, and config schema instead",
			path,
		)
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

type capContractBus struct{}

func (capContractBus) Publish(context.Context, string, []byte) error {
	return nil
}

func (capContractBus) Subscribe(
	context.Context,
	string,
	func(context.Context, *nats.Msg),
) (capcomponent.Subscription, error) {
	return contractSubscription{}, nil
}

type capContractPlanWriter struct{}

func (capContractPlanWriter) Apply(context.Context, capprojector.Plan) error {
	return nil
}

type adsbContractBus struct{}

func (adsbContractBus) Publish(context.Context, string, []byte) error {
	return nil
}

func (adsbContractBus) Subscribe(
	context.Context,
	string,
	func(context.Context, *nats.Msg),
) (adsbcomponent.Subscription, error) {
	return contractSubscription{}, nil
}

type adsbContractPlanWriter struct{}

func (adsbContractPlanWriter) Apply(context.Context, adsbprojector.Plan) error {
	return nil
}

type sapientContractBus struct{}

func (sapientContractBus) Publish(context.Context, string, []byte) error {
	return nil
}

func (sapientContractBus) Subscribe(
	context.Context,
	string,
	func(context.Context, *nats.Msg),
) (sapientcomponent.Subscription, error) {
	return contractSubscription{}, nil
}
