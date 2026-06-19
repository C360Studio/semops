package cop

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	copmodel "github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

func TestGraphProviderMapsMAVLinkEntities(t *testing.T) {
	now := time.Date(2026, 6, 19, 13, 0, 0, 0, time.UTC)
	observed := now.Add(-12 * time.Second)
	assetID := "c360.edge-compose.cop.mavlink.asset.system-42"
	trackID := "c360.edge-compose.cop.mavlink.track.system-42"
	requester := &fakeGraphSnapshotRequester{
		entities: map[string]graph.EntityState{
			assetID: {
				ID:        assetID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(assetID, copmodel.AssetName, "MAVLink system 42", observed),
					testTriple(assetID, copmodel.AssetKind, "mavlink-system", observed),
					testTriple(assetID, copmodel.AssetSource, "mavlink", observed),
					testTriple(assetID, copmodel.ProvenanceConfidence, 0.98, observed),
					testTriple(assetID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(assetID, copmodel.ProvenanceSourceRef, "mavlink://raw/udp/00000001", observed),
				},
			},
			trackID: {
				ID:        trackID,
				UpdatedAt: observed,
				Triples: []message.Triple{
					testTriple(trackID, copmodel.TrackNativeID, "mavlink.system.42.component.7", observed),
					testTriple(trackID, copmodel.TrackStatus, "active.armed", observed),
					testTriple(trackID, copmodel.TrackPosition, "POINT(-77.0000002 38.9000001)", observed),
					testTriple(trackID, copmodel.TrackVelocity, "NED_CMPS(321 -12 7)", observed),
					testTriple(trackID, copmodel.TrackSource, assetID, observed),
					testTriple(trackID, copmodel.TrackObservedAt, observed, observed),
					testTriple(trackID, copmodel.ProvenanceSource, "mavlink", observed),
					testTriple(trackID, copmodel.ProvenanceConfidence, 0.98, observed),
					testTriple(trackID, copmodel.ProvenanceObservedAt, observed, observed),
					testTriple(trackID, copmodel.ProvenanceSourceRef, "mavlink://raw/udp/00000002", observed),
				},
			},
		},
	}
	provider, err := NewGraphProvider(
		requester,
		WithGraphNow(func() time.Time { return now }),
		WithMAVLinkSystems("c360", "edge-compose", []int{42}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.Scenario != "phase-1-live-graph" {
		t.Fatalf("scenario = %q", snapshot.Scenario)
	}
	if snapshot.Summary.ActiveTracks != 1 || len(snapshot.Tracks) != 1 {
		t.Fatalf("track summary/list = %d/%d", snapshot.Summary.ActiveTracks, len(snapshot.Tracks))
	}
	track := snapshot.Tracks[0]
	if track.ID != trackID || track.Label != "UAS 42" || track.Status != "active.armed" {
		t.Fatalf("track = %+v", track)
	}
	if track.Position.Lat != 38.9000001 || track.Position.Lon != -77.0000002 {
		t.Fatalf("track position = %+v", track.Position)
	}
	if track.Provenance.Owner != copmodel.OwnerMAVLink || track.Provenance.SourceRef != "mavlink://raw/udp/00000002" {
		t.Fatalf("track provenance = %+v", track.Provenance)
	}
	if len(snapshot.Assets) != 1 || snapshot.Assets[0].Position == nil {
		t.Fatalf("asset missing position copied from source track: %+v", snapshot.Assets)
	}
	if snapshot.Assets[0].Position.Lat != track.Position.Lat || snapshot.Assets[0].Position.Lon != track.Position.Lon {
		t.Fatalf("asset position = %+v, want track position %+v", snapshot.Assets[0].Position, track.Position)
	}
	if snapshot.Feeds[0].Status != "live" {
		t.Fatalf("MAVLink feed status = %q", snapshot.Feeds[0].Status)
	}
	if got := requester.subjects; len(got) != 2 || got[0] != SubjectGraphQueryEntity || got[1] != SubjectGraphQueryEntity {
		t.Fatalf("query subjects = %+v", got)
	}
}

func TestGraphProviderFallsBackWhenNoGraphEntitiesExist(t *testing.T) {
	now := time.Date(2026, 6, 19, 14, 0, 0, 0, time.UTC)
	provider, err := NewGraphProvider(
		legacyNotFoundRequester{},
		WithGraphNow(func() time.Time { return now }),
		WithGraphFallback(staticSnapshotProvider{snapshot: Snapshot{
			GeneratedAt: now,
			Scenario:    "fixture-fallback",
			Summary:     Summary{ActiveTracks: 1},
		}}),
	)
	if err != nil {
		t.Fatalf("new graph provider: %v", err)
	}

	snapshot, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snapshot.Scenario != "fixture-fallback" {
		t.Fatalf("scenario = %q", snapshot.Scenario)
	}
}

type fakeGraphSnapshotRequester struct {
	entities map[string]graph.EntityState
	subjects []string
}

func (r *fakeGraphSnapshotRequester) Request(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	return r.request(ctx, subject, data, timeout)
}

func (r *fakeGraphSnapshotRequester) RequestClassified(
	ctx context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	return r.request(ctx, subject, data, timeout)
}

func (r *fakeGraphSnapshotRequester) request(
	_ context.Context,
	subject string,
	data []byte,
	_ time.Duration,
) ([]byte, error) {
	r.subjects = append(r.subjects, subject)
	var req map[string]string
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	entity, ok := r.entities[req["id"]]
	if !ok {
		return nil, errors.New("not found: " + req["id"])
	}
	return json.Marshal(entity)
}

type legacyNotFoundRequester struct{}

func (legacyNotFoundRequester) Request(
	context.Context,
	string,
	[]byte,
	time.Duration,
) ([]byte, error) {
	return []byte("error: not found: no such entity"), nil
}

type staticSnapshotProvider struct {
	snapshot Snapshot
}

func (p staticSnapshotProvider) Snapshot(context.Context) (Snapshot, error) {
	return p.snapshot, nil
}

func testTriple(subject, predicate string, object any, observed time.Time) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     "test",
		Timestamp:  observed,
		Confidence: 0.98,
	}
}
