package cap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/internal/copownership"
	capprojector "github.com/c360studio/semops/internal/projectors/cap"
	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const liveGraphNATSEnv = "SEMOPS_CAP_LIVE_GRAPH_NATS_URL"

func TestLiveGraphCAPBornFirstSmoke(t *testing.T) {
	natsURL := os.Getenv(liveGraphNATSEnv)
	if natsURL == "" {
		t.Skipf("set %s to run the live CAP graph smoke", liveGraphNATSEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := natsclient.NewClient(
		natsURL,
		natsclient.WithName("semops-cap-live-graph-smoke"),
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
	projector := capprojector.NewProjector(capprojector.Config{
		Org:         "c360",
		Platform:    platform,
		OwnerTokens: bindings.OwnerTokenMap(),
		TraceID:     "semops-cap-live-graph-smoke",
	})
	writer := capprojector.NewGraphWriter(client, capprojector.WithWriteTimeout(2*time.Second))

	alert, err := capcodec.Parse([]byte(sampleCAPAlert))
	if err != nil {
		t.Fatalf("parse sample CAP alert: %v", err)
	}
	plan, err := projector.ProjectAlert(alert, "cap://fixture/nws-demo-flood-warning")
	if err != nil {
		t.Fatalf("project CAP alert: %v", err)
	}
	if err := writer.Apply(ctx, plan); err != nil {
		assertNoMustExistFailure(t, err)
		t.Fatalf("write CAP create plan: %v", err)
	}
	projector.MarkBornForPlan(plan)

	alert.MsgType = "Update"
	updatePlan, err := projector.ProjectAlert(alert, "cap://fixture/nws-demo-flood-update")
	if err != nil {
		t.Fatalf("project CAP update: %v", err)
	}
	if err := writer.Apply(ctx, updatePlan); err != nil {
		assertNoMustExistFailure(t, err)
		t.Fatalf("write CAP update plan: %v", err)
	}

	hazardID := capprojector.EntityID("c360", platform, alert.Identifier)
	entities := pollPrefix(ctx, t, client, graphEntityPrefix("c360", platform, "cap", cop.EntityHazardArea))
	hazard := requireEntity(t, entities, hazardID)
	if hazard.MessageType.Key() != cop.CAPHazardEvidenceContract().MessageType {
		t.Fatalf("hazard message type = %q, want %q",
			hazard.MessageType.Key(), cop.CAPHazardEvidenceContract().MessageType)
	}
	requireTriple(t, hazard.Triples, cop.HazardSource, "cap")
	requireTriple(t, hazard.Triples, cop.ProvenanceSource, "cap")
	requireAnyTriple(t, hazard.Triples, cop.ProvenanceSourceRef, "cap://fixture/nws-demo-flood-update")
	requireStringTripleContaining(t, hazard.Triples, cop.HazardEvidence,
		`"message_type":"Update"`,
		`"event":"Flood Warning"`,
	)
	if hasPredicate(hazard.Triples, cop.HazardGeometry) ||
		hasPredicate(hazard.Triples, cop.HazardSeverity) ||
		hasPredicate(hazard.Triples, cop.HazardStatus) {
		t.Fatalf("CAP smoke must not write authoritative hazard predicates: %+v", hazard.Triples)
	}
}

func pollPrefix(
	ctx context.Context,
	t *testing.T,
	client *natsclient.Client,
	prefix string,
) []graph.EntityState {
	t.Helper()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		entities, err := queryPrefix(ctx, client, prefix)
		if err == nil && len(entities) > 0 {
			return entities
		}
		lastErr = err

		select {
		case <-ctx.Done():
			t.Fatalf("poll prefix %s: %v; last error: %v", prefix, ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func queryPrefix(
	ctx context.Context,
	client *natsclient.Client,
	prefix string,
) ([]graph.EntityState, error) {
	body, err := json.Marshal(map[string]any{"prefix": prefix, "limit": 25})
	if err != nil {
		return nil, err
	}
	response, err := client.Request(ctx, "graph.query.prefix", body, 2*time.Second)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Entities []graph.EntityState `json:"entities"`
		Error    string              `json:"error"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != "" {
		return nil, errors.New(envelope.Error)
	}
	return envelope.Entities, nil
}

func graphEntityPrefix(org, platform, system, entityType string) string {
	return strings.Join([]string{org, platform, "cop", system, entityType}, ".")
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

func requireAnyTriple(t *testing.T, triples []message.Triple, predicate string, want any) {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate && triple.Object == want {
			return
		}
	}
	t.Fatalf("missing %s object %#v in %+v", predicate, want, triples)
}

func requireEntity(t *testing.T, entities []graph.EntityState, id string) graph.EntityState {
	t.Helper()
	for _, entity := range entities {
		if entity.ID == id {
			return entity
		}
	}
	t.Fatalf("missing entity %s in prefix result %+v", id, entities)
	return graph.EntityState{}
}

func requireStringTripleContaining(
	t *testing.T,
	triples []message.Triple,
	predicate string,
	parts ...string,
) {
	t.Helper()
	seen := make([]string, 0)
	for _, triple := range triples {
		if triple.Predicate == predicate {
			value, ok := triple.Object.(string)
			if !ok {
				t.Fatalf("%s object = %#v, want string", predicate, triple.Object)
			}
			seen = append(seen, value)
			matched := true
			for _, part := range parts {
				if !strings.Contains(value, part) {
					matched = false
					break
				}
			}
			if matched {
				return
			}
		}
	}
	t.Fatalf("no %s string triple contained %v; saw %v", predicate, parts, seen)
}

func hasPredicate(triples []message.Triple, predicate string) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return true
		}
	}
	return false
}

const sampleCAPAlert = `<?xml version="1.0" encoding="UTF-8"?>
<alert xmlns="urn:oasis:names:tc:emergency:cap:1.2">
  <identifier>nws-demo-flood-warning</identifier>
  <sender>w-nws.webmaster@noaa.gov</sender>
  <sent>2026-06-19T15:04:05Z</sent>
  <status>Actual</status>
  <msgType>Alert</msgType>
  <source>NWS</source>
  <scope>Public</scope>
  <info>
    <language>en-US</language>
    <category>Met</category>
    <event>Flood Warning</event>
    <urgency>Immediate</urgency>
    <severity>Severe</severity>
    <certainty>Likely</certainty>
    <effective>2026-06-19T15:04:05Z</effective>
    <expires>2026-06-19T18:04:05Z</expires>
    <senderName>National Weather Service</senderName>
    <headline>Flood Warning issued for North Branch</headline>
    <description>Flooding is occurring near low crossings.</description>
    <instruction>Move to higher ground. Avoid flooded roadways.</instruction>
    <area>
      <areaDesc>North Branch</areaDesc>
      <polygon>38.895,-77.012 38.907,-77.011 38.908,-76.992 38.896,-76.991</polygon>
    </area>
  </info>
</alert>`
