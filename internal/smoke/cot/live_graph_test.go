package cot

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/internal/copownership"
	cotprojector "github.com/c360studio/semops/internal/projectors/cot"
	"github.com/c360studio/semops/internal/stack"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const liveGraphNATSEnv = "SEMOPS_COT_LIVE_GRAPH_NATS_URL"

func TestLiveGraphCoTBornFirstSmoke(t *testing.T) {
	natsURL := os.Getenv(liveGraphNATSEnv)
	if natsURL == "" {
		t.Skipf("set %s to run the live CoT graph smoke", liveGraphNATSEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := natsclient.NewClient(
		natsURL,
		natsclient.WithName("semops-cot-live-graph-smoke"),
		natsclient.WithTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer closeCancel()
		if err := client.Close(closeCtx); err != nil {
			t.Logf("close nats client: %v", err)
		}
	})

	ownerRegistry, err := ownership.EnsureBuckets(ctx, client, nil, nil)
	if err != nil {
		t.Fatalf("ensure ownership buckets: %v", err)
	}
	heartbeater := ownerRegistry.NewHeartbeater(ownership.HeartbeatInterval)
	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	t.Cleanup(heartbeatCancel)
	go heartbeater.Run(heartbeatCtx)

	bindings, err := copownership.RegisterFirstPhase(ctx, ownerRegistry, heartbeater)
	if err != nil {
		t.Fatalf("register first-phase COP ownership: %v", err)
	}

	platform := "smoke-" + time.Now().UTC().Format("20060102150405")
	adapter, err := stack.NewCoTAdapter(stack.CoTAdapterConfig{
		Source:       "live-graph-smoke",
		Org:          "c360",
		Platform:     platform,
		OwnerTokens:  bindings.OwnerTokenMap(),
		TraceID:      "semops-cot-live-graph-smoke",
		WriteTimeout: 2 * time.Second,
		Retry: natsclient.RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    50 * time.Millisecond,
			MaxBackoff:        500 * time.Millisecond,
			BackoffMultiplier: 2,
		},
	}, stack.CoTAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new cot adapter: %v", err)
	}

	rawEvents, err := cotcodec.MarshalEvents(cotcodec.SeedEvents(time.Now().UTC()))
	if err != nil {
		t.Fatalf("marshal seed events: %v", err)
	}
	for i, raw := range rawEvents {
		if _, err := adapter.IngestEvent(ctx, raw); err != nil {
			assertNoMustExistFailure(t, err)
			t.Fatalf("ingest cot event %d: %v", i+1, err)
		}
	}

	trackID := cotprojector.EntityID("c360", platform, cop.EntityTrack, "ANDROID-ALPHA")
	assetID := cotprojector.EntityID("c360", platform, cop.EntityAsset, "ANDROID-ALPHA")
	taskID := cotprojector.EntityID("c360", platform, cop.EntityTask, "MARKER-NORTH-GATE")
	advisoryID := cotprojector.EntityID("c360", platform, cop.EntityAdvisory, "CHAT-ALPHA-1")

	track := pollEntity(ctx, t, client, trackID)
	task := pollEntity(ctx, t, client, taskID)
	advisory := pollEntity(ctx, t, client, advisoryID)

	if track.MessageType.Key() != cop.TAKTrackContract().MessageType {
		t.Fatalf("track message type = %q, want %q", track.MessageType.Key(), cop.TAKTrackContract().MessageType)
	}
	if task.MessageType.Key() != cop.TAKTaskContract().MessageType {
		t.Fatalf("task message type = %q, want %q", task.MessageType.Key(), cop.TAKTaskContract().MessageType)
	}
	if advisory.MessageType.Key() != cop.TAKAdvisoryContract().MessageType {
		t.Fatalf("advisory message type = %q, want %q",
			advisory.MessageType.Key(), cop.TAKAdvisoryContract().MessageType)
	}
	requireTriple(t, track.Triples, cop.TrackSource, assetID)
	requireTriple(t, track.Triples, cop.TrackPosition, "POINT(-77.0350000 38.8920000)")
	requireTriple(t, task.Triples, cop.TaskPosition, "POINT(-77.0380000 38.8940000)")
	requireTriple(t, task.Triples, cop.TaskDescription, "checkpoint")
	requireTriple(t, advisory.Triples, cop.AdvisoryText, "hold at checkpoint")
	requireTriple(t, advisory.Triples, cop.AdvisorySender, "ANDROID-ALPHA")
	assertCoTOwnership(t, ctx, ownerRegistry, trackID, taskID, advisoryID)

	health := adapter.Health()
	if !health.Ready || health.WriteErrors != 0 || health.ProjectionDrops != 0 {
		t.Fatalf("adapter health = %+v", health)
	}
}

func assertCoTOwnership(
	t *testing.T,
	ctx context.Context,
	registry *ownership.Registry,
	trackID string,
	taskID string,
	advisoryID string,
) {
	t.Helper()

	for _, check := range []struct {
		entityID  string
		predicate string
	}{
		{entityID: trackID, predicate: cop.TrackPosition},
		{entityID: taskID, predicate: cop.TaskPosition},
		{entityID: advisoryID, predicate: cop.AdvisoryText},
	} {
		owner, ok, err := registry.OwnerOf(ctx, check.entityID, check.predicate)
		if err != nil {
			t.Fatalf("lookup owner for %s/%s: %v", check.entityID, check.predicate, err)
		}
		if !ok || owner != cop.OwnerTAK {
			t.Fatalf("owner for %s/%s = %q,%v, want %q,true",
				check.entityID, check.predicate, owner, ok, cop.OwnerTAK)
		}
	}
	edge, ok, err := registry.ForeignEdgeClaimFor(ctx, cop.TAKTrackContract().MessageType, cop.TrackSource)
	if err != nil {
		t.Fatalf("lookup TAK foreign-edge claim: %v", err)
	}
	if !ok || edge.Owner != cop.OwnerTAK || edge.Mode != ownership.EdgeStrict {
		t.Fatalf("foreign-edge claim = %+v, ok=%v, want owner %q strict", edge, ok, cop.OwnerTAK)
	}
}

func pollEntity(
	ctx context.Context,
	t *testing.T,
	client *natsclient.Client,
	entityID string,
) graph.EntityState {
	t.Helper()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		entity, err := queryEntity(ctx, client, entityID)
		if err == nil {
			return entity
		}
		lastErr = err

		select {
		case <-ctx.Done():
			t.Fatalf("poll entity %s: %v; last error: %v", entityID, ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func queryEntity(
	ctx context.Context,
	client *natsclient.Client,
	entityID string,
) (graph.EntityState, error) {
	body, err := json.Marshal(map[string]string{"id": entityID})
	if err != nil {
		return graph.EntityState{}, err
	}
	response, err := client.Request(ctx, "graph.query.entity", body, 2*time.Second)
	if err != nil {
		return graph.EntityState{}, err
	}
	var entity graph.EntityState
	if err := json.Unmarshal(response, &entity); err != nil {
		return graph.EntityState{}, err
	}
	if entity.ID == "" {
		return graph.EntityState{}, errors.New("empty entity response")
	}
	return entity, nil
}

func assertNoMustExistFailure(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	for _, forbidden := range []string{"entity_not_found", "foreign_edge_dropped"} {
		if strings.Contains(msg, forbidden) {
			t.Fatalf("must-exist compliance failure %q in error: %v", forbidden, err)
		}
	}
}

func requireTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate {
			if triple.Object != want {
				t.Fatalf("%s object = %#v, want %#v", predicate, triple.Object, want)
			}
			return
		}
	}
	t.Fatalf("missing predicate %q in %+v", predicate, triples)
}
