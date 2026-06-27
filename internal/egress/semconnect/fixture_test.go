package semconnect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunReadSideFixtureProjectsSnapshotThroughSemConnectBoundary(t *testing.T) {
	observed := time.Date(2026, 6, 27, 14, 15, 0, 0, time.UTC)
	var captured []capturedHTTPRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Errorf("decode body %s: %v", string(body), err)
		}
		captured = append(captured, capturedHTTPRequest{
			Method:      r.Method,
			Path:        r.URL.Path,
			ContentType: r.Header.Get("Content-Type"),
			UserAgent:   r.Header.Get("User-Agent"),
			Body:        decoded,
		})
		w.Header().Set("Location", r.URL.Path+"/created")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	result, err := RunReadSideFixture(context.Background(), server.URL+"/csapi",
		WithFixtureObservedAt(observed),
		WithFixtureExecuteOptions(WithUserAgent("semops-fixture-test")),
	)
	if err != nil {
		t.Fatalf("RunReadSideFixture: %v", err)
	}
	if result.FixtureID != DefaultReadSideFixtureID ||
		result.Status != FixtureStatusPassed ||
		result.CatalogCounts.Systems != 1 ||
		result.CatalogCounts.Deployments != 1 ||
		result.CatalogCounts.Datastreams != 1 ||
		result.CatalogCounts.Observations != 1 ||
		result.CatalogCounts.SystemEvents != 1 {
		t.Fatalf("fixture result = %+v", result)
	}
	if len(captured) != 5 {
		t.Fatalf("captured requests = %d, want 5: %+v", len(captured), captured)
	}
	wantPaths := []string{
		"/csapi/systems",
		"/csapi/deployments",
		"/csapi/datastreams",
		"/csapi/datastreams/" + result.Plan.IDs.Datastreams["c360.edge.cop.mavlink.track.system-42/datastreams/position"] + "/observations",
		"/csapi/systemEvents",
	}
	for i, wantPath := range wantPaths {
		if captured[i].Method != http.MethodPost || captured[i].Path != wantPath || captured[i].UserAgent != "semops-fixture-test" {
			t.Fatalf("captured[%d] = %+v, want POST %s", i, captured[i], wantPath)
		}
	}

	systemBody := captured[0].Body["properties"].(map[string]any)
	if systemBody["name"] != "MAVLink system 42" {
		t.Fatalf("system body = %#v", systemBody)
	}
	observationBody := captured[3].Body
	resultBody := observationBody["result"].(map[string]any)
	if observationBody["observedProperty"] != "https://c360.studio/semops/cop/observed-property/position" ||
		resultBody["semops_entity_type"] != "track" ||
		resultBody["status"] != "active.armed" {
		t.Fatalf("observation body = %#v", observationBody)
	}
	if len(result.Catalog.DeferredSurfaces) == 0 {
		t.Fatal("fixture result should retain deferred command/write-side surfaces")
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("fixture evidence should marshal as JSON: %v", err)
	}
}

func TestRunReadSideFixtureReturnsPartialEvidenceOnSemConnectFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/datastreams") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad datastream"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	result, err := RunReadSideFixture(context.Background(), server.URL, WithFixtureObservedAt(time.Date(2026, 6, 27, 14, 15, 0, 0, time.UTC)))
	if err == nil {
		t.Fatal("expected SemConnect failure")
	}
	if result.Status != FixtureStatusFailed ||
		!strings.Contains(result.Error, "bad datastream") ||
		len(result.Execution.Requests) != 3 {
		t.Fatalf("partial evidence = %+v, err=%v", result, err)
	}
}
