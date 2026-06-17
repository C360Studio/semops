package mavlink

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/internal/copownership"
	"github.com/c360studio/semops/internal/stack"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/ownership"
)

const (
	liveGraphNATSEnv    = "SEMOPS_MAVLINK_LIVE_GRAPH_NATS_URL"
	liveGraphMetricsEnv = "SEMOPS_MAVLINK_LIVE_GRAPH_METRICS_URL"
)

func TestLiveGraphMAVLinkBornFirstSmoke(t *testing.T) {
	natsURL := os.Getenv(liveGraphNATSEnv)
	if natsURL == "" {
		t.Skipf("set %s to run the live MAVLink graph smoke", liveGraphNATSEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := natsclient.NewClient(
		natsURL,
		natsclient.WithName("semops-mavlink-live-graph-smoke"),
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
	metricsBefore, hasMetrics := maybeScrapeMetrics(ctx, t)

	platform := "smoke-" + time.Now().UTC().Format("20060102150405")
	adapter, err := stack.NewMAVLinkAdapter(stack.MAVLinkAdapterConfig{
		Source:       "live-graph-smoke",
		Org:          "c360",
		Platform:     platform,
		OwnerTokens:  bindings.OwnerTokenMap(),
		TraceID:      "semops-mavlink-live-graph-smoke",
		WriteTimeout: 2 * time.Second,
		Retry: natsclient.RetryConfig{
			MaxRetries:        5,
			InitialBackoff:    50 * time.Millisecond,
			MaxBackoff:        500 * time.Millisecond,
			BackoffMultiplier: 2,
		},
	}, stack.MAVLinkAdapterDeps{NATS: client})
	if err != nil {
		t.Fatalf("new mavlink adapter: %v", err)
	}

	generator := mavcodec.NewGenerator(42, 7)
	heartbeat, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	position, err := generator.GenerateGlobalPosition(mavcodec.PositionMessage{
		Lat: 389000001,
		Lon: -770000002,
		Vx:  321,
		Vy:  -12,
		Vz:  7,
	})
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}

	if _, err := adapter.IngestFrame(ctx, heartbeat); err != nil {
		assertNoMustExistFailure(t, err)
		t.Fatalf("ingest heartbeat: %v", err)
	}
	if _, err := adapter.IngestFrame(ctx, position); err != nil {
		assertNoMustExistFailure(t, err)
		t.Fatalf("ingest position: %v", err)
	}

	assetID := "c360." + platform + ".cop.mavlink.asset.system-42"
	trackID := "c360." + platform + ".cop.mavlink.track.system-42"
	assertMAVLinkOwnership(t, ctx, ownerRegistry, trackID)
	asset := pollEntity(ctx, t, client, assetID)
	track := pollEntity(ctx, t, client, trackID)

	if asset.MessageType.Key() != cop.SourceAssetContract().MessageType {
		t.Fatalf("asset message type = %q, want %q",
			asset.MessageType.Key(), cop.SourceAssetContract().MessageType)
	}
	if track.MessageType.Key() != cop.MAVLinkTrackContract().MessageType {
		t.Fatalf("track message type = %q, want %q",
			track.MessageType.Key(), cop.MAVLinkTrackContract().MessageType)
	}
	requireTriple(t, track.Triples, cop.TrackSource, assetID)
	requireTriple(t, track.Triples, cop.TrackPosition, "POINT(-77.0000002 38.9000001)")
	if hasMetrics {
		metricsAfter := scrapeMetrics(ctx, t)
		assertNoGraphIngestCounterDeltas(t, metricsBefore, metricsAfter)
	}
}

func assertMAVLinkOwnership(
	t *testing.T,
	ctx context.Context,
	registry *ownership.Registry,
	trackID string,
) {
	t.Helper()

	owner, ok, err := registry.OwnerOf(ctx, trackID, cop.TrackPosition)
	if err != nil {
		t.Fatalf("lookup MAVLink track owner: %v", err)
	}
	if !ok || owner != cop.OwnerMAVLink {
		t.Fatalf("MAVLink track owner = %q,%v, want %q,true", owner, ok, cop.OwnerMAVLink)
	}
	edge, ok, err := registry.ForeignEdgeClaimFor(ctx, cop.MAVLinkTrackContract().MessageType, cop.TrackSource)
	if err != nil {
		t.Fatalf("lookup MAVLink foreign-edge claim: %v", err)
	}
	if !ok {
		t.Fatalf("missing foreign-edge claim for %s/%s", cop.MAVLinkTrackContract().MessageType, cop.TrackSource)
	}
	if edge.Owner != cop.OwnerMAVLink || edge.Mode != ownership.EdgeStrict {
		t.Fatalf("foreign-edge claim = %+v, want owner %q strict", edge, cop.OwnerMAVLink)
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

type metricSample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

type metricSnapshot []metricSample

func maybeScrapeMetrics(ctx context.Context, t *testing.T) (metricSnapshot, bool) {
	t.Helper()
	if os.Getenv(liveGraphMetricsEnv) == "" {
		return nil, false
	}
	return scrapeMetrics(ctx, t), true
}

func scrapeMetrics(ctx context.Context, t *testing.T) metricSnapshot {
	t.Helper()
	url := os.Getenv(liveGraphMetricsEnv)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("create metrics request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("scrape metrics %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("scrape metrics status = %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	snapshot, err := parsePrometheusMetrics(string(body))
	if err != nil {
		t.Fatalf("parse metrics: %v", err)
	}
	return snapshot
}

func parsePrometheusMetrics(body string) (metricSnapshot, error) {
	var snapshot metricSnapshot
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		name, labels, err := parseMetricIdentity(fields[0])
		if err != nil {
			return nil, err
		}
		snapshot = append(snapshot, metricSample{Name: name, Labels: labels, Value: value})
	}
	return snapshot, nil
}

func parseMetricIdentity(identity string) (string, map[string]string, error) {
	open := strings.IndexByte(identity, '{')
	if open == -1 {
		return identity, nil, nil
	}
	if !strings.HasSuffix(identity, "}") {
		return "", nil, errors.New("metric labels missing closing brace")
	}
	labels := map[string]string{}
	for _, part := range strings.Split(identity[open+1:len(identity)-1], ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return "", nil, errors.New("metric label missing equals")
		}
		labels[key] = strings.Trim(value, `"`)
	}
	return identity[:open], labels, nil
}

func (s metricSnapshot) sum(name string, labels map[string]string) float64 {
	var total float64
	for _, sample := range s {
		if sample.Name != name {
			continue
		}
		if labelsMatch(sample.Labels, labels) {
			total += sample.Value
		}
	}
	return total
}

func labelsMatch(got, want map[string]string) bool {
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	return true
}

func assertNoGraphIngestCounterDeltas(t *testing.T, before, after metricSnapshot) {
	t.Helper()
	for _, check := range []struct {
		name   string
		labels map[string]string
	}{
		{
			name: "semstreams_graph_ingest_owner_lease_mismatch_total",
			labels: map[string]string{
				"message_type": cop.MAVLinkTrackContract().MessageType,
			},
		},
		{
			name: "semstreams_graph_ingest_owner_lease_mismatch_total",
			labels: map[string]string{
				"message_type": cop.SourceAssetContract().MessageType,
			},
		},
		{
			name: "semstreams_graph_ingest_foreign_edge_unclaimed_total",
			labels: map[string]string{
				"message_type": cop.MAVLinkTrackContract().MessageType,
				"predicate":    cop.TrackSource,
			},
		},
		{
			name: "semstreams_graph_ingest_foreign_edge_dropped_total",
			labels: map[string]string{
				"message_type": cop.MAVLinkTrackContract().MessageType,
				"predicate":    cop.TrackSource,
			},
		},
		{
			name: "semstreams_graph_ingest_indexing_profile_default_total",
			labels: map[string]string{
				"message_type": cop.MAVLinkTrackContract().MessageType,
			},
		},
		{
			name: "semstreams_graph_ingest_indexing_profile_default_total",
			labels: map[string]string{
				"message_type": cop.SourceAssetContract().MessageType,
			},
		},
	} {
		if delta := after.sum(check.name, check.labels) - before.sum(check.name, check.labels); delta != 0 {
			t.Fatalf("%s%v delta = %v, want 0", check.name, check.labels, delta)
		}
	}
}

func TestParsePrometheusMetrics(t *testing.T) {
	snapshot, err := parsePrometheusMetrics(`
# HELP semstreams_graph_ingest_owner_lease_mismatch_total help
semstreams_graph_ingest_owner_lease_mismatch_total{message_type="mav-track",predicate="cop.track.position"} 2
semstreams_graph_ingest_owner_lease_mismatch_total{message_type="mav-track",predicate="cop.track.velocity"} 3
semstreams_graph_ingest_indexing_profile_default_total{message_type="unknown"} 4
`)
	if err != nil {
		t.Fatalf("parse metrics: %v", err)
	}
	got := snapshot.sum("semstreams_graph_ingest_owner_lease_mismatch_total", map[string]string{
		"message_type": "mav-track",
	})
	if got != 5 {
		t.Fatalf("summed owner mismatch = %v, want 5", got)
	}
	unknown := snapshot.sum("semstreams_graph_ingest_indexing_profile_default_total", map[string]string{
		"message_type": "unknown",
	})
	if unknown != 4 {
		t.Fatalf("unknown indexing defaults = %v, want 4", unknown)
	}
}
