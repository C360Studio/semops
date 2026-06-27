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

	readmodel "github.com/c360studio/semops/internal/egress/csapi"
)

type capturedHTTPRequest struct {
	Method      string
	Path        string
	ContentType string
	UserAgent   string
	Body        map[string]any
}

func TestExecuteReadSidePlanPostsToSemConnectHTTPBoundary(t *testing.T) {
	plan := executableTestPlan(t)
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
		w.Header().Set("Content-Type", MediaJSON)
		w.Header().Set("Location", r.URL.Path+"/created")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"created"}`))
	}))
	defer server.Close()

	result, err := ExecuteReadSidePlan(context.Background(), server.URL+"/csapi", plan, WithUserAgent("semops-test"))
	if err != nil {
		t.Fatalf("ExecuteReadSidePlan: %v", err)
	}
	if len(result.Requests) != len(plan.Requests) || len(captured) != len(plan.Requests) {
		t.Fatalf("request counts: result=%d captured=%d plan=%d", len(result.Requests), len(captured), len(plan.Requests))
	}
	wantPaths := []string{
		"/csapi/systems",
		"/csapi/deployments",
		"/csapi/datastreams",
		"/csapi/datastreams/" + plan.IDs.Datastreams["semops.track.42/datastreams/position"] + "/observations",
		"/csapi/systemEvents",
	}
	for i, wantPath := range wantPaths {
		if captured[i].Method != http.MethodPost || captured[i].Path != wantPath || captured[i].UserAgent != "semops-test" {
			t.Fatalf("captured[%d] = %+v, want POST %s", i, captured[i], wantPath)
		}
		if result.Requests[i].StatusCode != http.StatusCreated || result.Requests[i].Location == "" {
			t.Fatalf("result[%d] = %+v", i, result.Requests[i])
		}
	}
	if captured[0].ContentType != MediaJSON ||
		captured[1].ContentType != MediaJSON ||
		captured[2].ContentType != MediaJSON ||
		captured[3].ContentType != MediaOMS ||
		captured[4].ContentType != MediaJSON {
		t.Fatalf("content types = %#v", []string{
			captured[0].ContentType,
			captured[1].ContentType,
			captured[2].ContentType,
			captured[3].ContentType,
			captured[4].ContentType,
		})
	}

	systemProps := captured[0].Body["properties"].(map[string]any)
	if systemProps["uid"] == "" || systemProps["name"] != "MAVLink system 42" {
		t.Fatalf("system properties = %#v", systemProps)
	}
	streamSystem := captured[2].Body["system"]
	if streamSystem != plan.IDs.Systems["semops.system.42"] {
		t.Fatalf("datastream system = %q, want %q", streamSystem, plan.IDs.Systems["semops.system.42"])
	}
	if captured[3].Body["observedProperty"] != "https://c360.studio/semops/cop/observed-property/position" {
		t.Fatalf("observation body = %#v", captured[3].Body)
	}
	eventPayload := captured[4].Body["payload"].(map[string]any)
	if eventPayload["status"] != "active" {
		t.Fatalf("event payload = %#v", eventPayload)
	}
}

func TestExecuteReadSidePlanReturnsPartialResultsOnFailure(t *testing.T) {
	plan := executableTestPlan(t)
	var seen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		if strings.Contains(r.URL.Path, "/datastreams") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad datastream"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	result, err := ExecuteReadSidePlan(context.Background(), server.URL, plan)
	if err == nil {
		t.Fatal("expected SemConnect failure")
	}
	if !strings.Contains(err.Error(), "bad datastream") {
		t.Fatalf("error = %v", err)
	}
	if seen != 3 {
		t.Fatalf("seen requests = %d, want stop after failing datastream", seen)
	}
	if len(result.Requests) != 3 ||
		result.Requests[0].StatusCode != http.StatusCreated ||
		result.Requests[2].StatusCode != http.StatusBadRequest {
		t.Fatalf("partial result = %+v", result)
	}
	if result.FinishedAt.IsZero() {
		t.Fatal("partial result should record FinishedAt")
	}
}

func TestExecuteReadSidePlanValidatesBaseURL(t *testing.T) {
	_, err := ExecuteReadSidePlan(context.Background(), "localhost:8080", ReadSidePlan{})
	if err == nil {
		t.Fatal("expected base URL validation error")
	}
}

func executableTestPlan(t *testing.T) ReadSidePlan {
	t.Helper()
	observed := time.Date(2026, 6, 26, 16, 30, 0, 0, time.UTC)
	catalog := readmodel.Catalog{
		GeneratedAt: observed,
		ClaimScope:  readmodel.ClaimScopeReadSideEgress,
		Systems: []readmodel.System{{
			ID:          "semops.system.42",
			Name:        "MAVLink system 42",
			Description: "SemOps MAVLink system",
			Source:      "mavlink",
			Provenance:  readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://system/42", ObservedAt: observed},
		}},
		Deployments: []readmodel.Deployment{{
			ID:         "semops.system.42/deployments/current",
			SystemID:   "semops.system.42",
			Name:       "MAVLink system 42 deployment",
			Source:     "mavlink",
			ObservedAt: observed,
			Provenance: readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://system/42", ObservedAt: observed},
		}},
		Datastreams: []readmodel.Datastream{{
			ID:               "semops.track.42/datastreams/position",
			SystemID:         "semops.system.42",
			Name:             "MAVLink system 42 position",
			ObservedProperty: "position",
			ResultType:       "GeoJSON Point",
			Source:           "mavlink",
			Provenance:       readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://track/42", ObservedAt: observed},
		}},
		Observations: []readmodel.Observation{{
			ID:             "semops.track.42/observations/current-position",
			DatastreamID:   "semops.track.42/datastreams/position",
			SystemID:       "semops.system.42",
			PhenomenonTime: observed,
			Result:         map[string]any{"status": "active"},
			Source:         "mavlink",
			Provenance:     readmodel.Provenance{Owner: "semops.feed.mavlink", SourceRef: "mavlink://track/42", ObservedAt: observed},
		}},
		SystemEvents: []readmodel.SystemEvent{{
			ID:         "semops.alert.track-freshness",
			SystemID:   "semops.system.42",
			EventType:  "cop.alert",
			Message:    "Track freshness nominal",
			Status:     "active",
			ObservedAt: observed,
		}},
	}
	plan, err := BuildReadSidePlan(catalog)
	if err != nil {
		t.Fatalf("BuildReadSidePlan: %v", err)
	}
	return plan
}
