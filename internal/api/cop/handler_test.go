package cop

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerServesSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	handler, err := NewHandler(NewFixtureProvider(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/snapshot", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var snapshot Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.GeneratedAt != now {
		t.Fatalf("generated at = %s, want %s", snapshot.GeneratedAt, now)
	}
	if snapshot.Summary.ActiveTracks != 2 || len(snapshot.Tracks) != 2 {
		t.Fatalf("tracks summary/list = %d/%d, want 2/2", snapshot.Summary.ActiveTracks, len(snapshot.Tracks))
	}
	if snapshot.Summary.ActiveTasks != 1 || snapshot.Summary.ActiveAdvisories != 1 {
		t.Fatalf("TAK summary = %+v", snapshot.Summary)
	}
	if snapshot.Tracks[0].Position.Lat == 0 || snapshot.Tracks[0].Position.Lon == 0 {
		t.Fatalf("track position missing: %+v", snapshot.Tracks[0].Position)
	}
	if snapshot.Tracks[0].Provenance.Owner != "semops.feed.mavlink" {
		t.Fatalf("track owner = %q", snapshot.Tracks[0].Provenance.Owner)
	}
}

func TestHandlerReportsProviderFailure(t *testing.T) {
	handler, err := NewHandler(failingProvider{})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/snapshot", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandlerServesRuntimeSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 21, 17, 30, 0, 0, time.UTC)
	handler, err := NewHandler(
		NewFixtureProvider(nil),
		WithClock(func() time.Time { return now }),
		WithRuntimeProvider(runtimeProviderStub{}),
	)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/cop/runtime", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	var snapshot RuntimeSnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode runtime snapshot: %v", err)
	}
	if snapshot.GeneratedAt != now {
		t.Fatalf("generated at = %s, want %s", snapshot.GeneratedAt, now)
	}
	if snapshot.Feeds == nil || snapshot.Components == nil {
		t.Fatalf("runtime snapshot slices should be JSON arrays: %+v", snapshot)
	}
}

func TestHandlerHealthz(t *testing.T) {
	handler, err := NewHandler(NewFixtureProvider(nil))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

type failingProvider struct{}

func (failingProvider) Snapshot(context.Context) (Snapshot, error) {
	return Snapshot{}, errors.New("snapshot unavailable")
}
