package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
)

const (
	trackEntityPattern = "c360.*.cop.*.track.*"
	trackSourcePattern = "c360.*.cop.*.source.*"
	trackOwner         = "semops.cop.track.mavlink"
)

func TestCurrentStateTrackProjectionUsesModernSemStreamsContracts(t *testing.T) {
	messageType := message.Type{
		Domain:   "semops",
		Category: "track",
		Version:  "v1",
	}

	contract := projection.Contract{
		Name:            "semops.cop.track.current-state",
		MessageType:     messageType.Key(),
		EntityPattern:   trackEntityPattern,
		IndexingProfile: "signal",
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
			Predicates: []string{
				"cop.track.position",
				"cop.track.velocity",
				"cop.track.status",
				"cop.track.updated_at",
			},
		}},
		ForeignEdges: []projection.ForeignEdge{{
			Predicate:     "cop.track.source",
			Mode:          ownership.EdgeStrict,
			TargetPattern: trackSourcePattern,
		}},
	}

	if err := contract.Validate(); err != nil {
		t.Fatalf("projection contract should validate: %v", err)
	}

	registration, err := projection.Derive(trackOwner, contract)
	if err != nil {
		t.Fatalf("projection contract should derive ownership: %v", err)
	}
	if registration.Owner != trackOwner {
		t.Fatalf("registration owner = %q, want %q", registration.Owner, trackOwner)
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
		Predicate:  "cop.track.position",
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
	if update.AddTriples[0].Predicate != "cop.track.position" {
		t.Fatalf("update predicate = %q, want cop.track.position", update.AddTriples[0].Predicate)
	}
}

func TestAdapterSkeletonUsesCurrentComponentLifecycleShape(t *testing.T) {
	var _ component.LifecycleComponent = (*adapterSkeleton)(nil)

	comp := &adapterSkeleton{}
	if err := comp.Initialize(); err != nil {
		t.Fatalf("initialize should be a no-op in the skeleton: %v", err)
	}
	if err := comp.Start(context.Background()); err != nil {
		t.Fatalf("start should accept caller context: %v", err)
	}
	if err := comp.Stop(time.Second); err != nil {
		t.Fatalf("stop should accept a timeout: %v", err)
	}
}

type adapterSkeleton struct{}

func (adapterSkeleton) Meta() component.Metadata {
	return component.Metadata{
		Name:        "semops-adapter-mavlink",
		Type:        "input",
		Description: "MAVLink boundary adapter skeleton for SemOps COP feeds",
		Version:     "v0",
	}
}

func (adapterSkeleton) InputPorts() []component.Port {
	return nil
}

func (adapterSkeleton) OutputPorts() []component.Port {
	return nil
}

func (adapterSkeleton) ConfigSchema() component.ConfigSchema {
	return component.ConfigSchema{}
}

func (adapterSkeleton) Health() component.HealthStatus {
	return component.HealthStatus{Healthy: true, Status: "not-started"}
}

func (adapterSkeleton) DataFlow() component.FlowMetrics {
	return component.FlowMetrics{}
}

func (adapterSkeleton) Initialize() error {
	return nil
}

func (adapterSkeleton) Start(context.Context) error {
	return nil
}

func (adapterSkeleton) Stop(time.Duration) error {
	return nil
}
