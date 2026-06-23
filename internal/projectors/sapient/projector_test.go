package sapient

import (
	"strings"
	"testing"
	"time"

	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectorCreatesSAPIENTTrackForAbsoluteLocationDetection(t *testing.T) {
	msg := parseMessage(t, absoluteDetectionJSON)
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "sapient-contract-001",
	})

	plan, err := projector.ProjectMessage(msg, "sapient://raw/fixture/detection-001")
	if err != nil {
		t.Fatalf("project SAPIENT detection: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want track birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])
	if create.Entity.ID != "c360.edge.cop.sapient.track.01ggyfbaxh4vyrqyex7s3xgk3h" {
		t.Fatalf("track id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.SAPIENTTrackContract().MessageType {
		t.Fatalf("track message type = %q", create.Entity.MessageType.Key())
	}
	if create.Entity.UpdatedAt != sampleObservedAt() {
		t.Fatalf("updated_at = %s, want %s", create.Entity.UpdatedAt, sampleObservedAt())
	}
	if create.IndexingProfile != cop.SAPIENTTrackContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.OwnerToken != "semops.feed.sapient#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "sapient-contract-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	if hasPredicate(create.Triples, cop.TrackSource) {
		t.Fatal("SAPIENT detection projection must not emit association/source foreign edges")
	}
	requireTriple(t, create.Triples, cop.TrackNativeID, "sapient.object.01ggyfbaxh4vyrqyex7s3xgk3h.node.a8654cdf-4328-47de-81fa-c495589e30c8.report.01ggyfbaxgdg7agahrz6xsny12")
	requireTriple(t, create.Triples, cop.TrackStatus, "active.detection.teststate")
	requireTriple(t, create.Triples, cop.TrackObservedAt, sampleObservedAt())
	requireTriple(t, create.Triples, cop.TrackPosition, "POINT(-1.8223767 51.1739726)")
	requireTriple(t, create.Triples, cop.ProvenanceSource, "sapient")
	requireTriple(t, create.Triples, cop.ProvenanceConfidence, 0.91)
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "sapient://raw/fixture/detection-001")
}

func TestProjectorUpdatesKnownSAPIENTDetectionWithoutRebirth(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	msg := parseMessage(t, absoluteDetectionJSON)

	birth, err := projector.ProjectMessage(msg, "sapient://raw/fixture/detection-001")
	if err != nil {
		t.Fatalf("project birth: %v", err)
	}
	if marked := projector.MarkBornForPlan(birth); marked != 1 {
		t.Fatalf("marked births = %d, want 1", marked)
	}

	next := parseMessage(t, strings.Replace(absoluteDetectionJSON, `"x": -1.82237671048`, `"x": -1.81237671048`, 1))
	plan, err := projector.ProjectMessage(next, "sapient://raw/fixture/detection-002")
	if err != nil {
		t.Fatalf("project update: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want update", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.sapient.track.01ggyfbaxh4vyrqyex7s3xgk3h" {
		t.Fatalf("update entity id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.SAPIENTTrackContract().IndexingProfile {
		t.Fatalf("update indexing profile = %q", update.IndexingProfile)
	}
	requireTriple(t, update.AddTriples, cop.TrackPosition, "POINT(-1.8123767 51.1739726)")
}

func TestProjectorRejectsAmbiguousSAPIENTCoordinateSystems(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "utm location",
			body: strings.Replace(absoluteDetectionJSON, CoordinateSystemLatLngDegM, "LOCATION_COORDINATE_SYSTEM_UTM_M", 1),
			want: "coordinate_system",
		},
		{
			name: "range bearing",
			body: rangeBearingDetectionJSON,
			want: "rangeBearing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProjector(Config{}).ProjectMessage(parseMessage(t, tt.body), "sapient://raw/fixture/rejected")
			if err == nil {
				t.Fatal("expected projection rejection")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestProjectorIgnoresNonDetectionMessages(t *testing.T) {
	msg, err := sapientcodec.ParseJSONMessage([]byte(taskAckJSON))
	if err != nil {
		t.Fatalf("parse task ack: %v", err)
	}
	plan, err := NewProjector(Config{}).ProjectMessage(msg, "sapient://raw/fixture/task-ack")
	if err != nil {
		t.Fatalf("project task ack: %v", err)
	}
	if len(plan.Mutations) != 0 {
		t.Fatalf("mutations = %d, want none for non-detection preflight message", len(plan.Mutations))
	}
}

func parseMessage(t *testing.T, body string) sapientcodec.Message {
	t.Helper()
	msg, err := sapientcodec.ParseJSONMessage([]byte(body))
	if err != nil {
		t.Fatalf("parse SAPIENT fixture: %v", err)
	}
	return msg
}

func sampleObservedAt() time.Time {
	return time.Date(2023, 7, 7, 12, 44, 17, 27638700, time.UTC)
}

func requireCreate(t *testing.T, mutation Mutation) graph.CreateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationCreate {
		t.Fatalf("mutation kind = %q, want create", mutation.Kind)
	}
	if mutation.Create.Entity == nil {
		t.Fatal("create entity is nil")
	}
	return mutation.Create
}

func requireUpdate(t *testing.T, mutation Mutation) graph.UpdateEntityWithTriplesRequest {
	t.Helper()
	if mutation.Kind != MutationUpdate {
		t.Fatalf("mutation kind = %q, want update", mutation.Kind)
	}
	if mutation.Update.Entity == nil {
		t.Fatal("update entity is nil")
	}
	return mutation.Update
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

func hasPredicate(triples []message.Triple, predicate string) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return true
		}
	}
	return false
}

func testOwnerTokens(incarnation string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerSAPIENT: ownership.ExpectedOwnerToken(cop.OwnerSAPIENT, incarnation),
	}
}

const absoluteDetectionJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "detectionReport": {
    "reportId": "01GGYFBAXGDG7AGAHRZ6XSNY12",
    "objectId": "01GGYFBAXH4VYRQYEX7S3XGK3H",
    "taskId": "01GGYFBAXHNV9DN0N74DFX2952",
    "state": "TestState",
    "location": {
      "x": -1.82237671048,
      "y": 51.1739726374,
      "z": 788,
      "coordinateSystem": "LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M",
      "datum": "LOCATION_DATUM_WGS84_E"
    },
    "detectionConfidence": 0.91
  }
}`

const rangeBearingDetectionJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "destinationId": "a8654cdf-4328-47de-81fa-c495589e30c9",
  "detectionReport": {
    "reportId": "01GGYFBAXGDG7AGAHRZ6XSNY12",
    "objectId": "01GGYFBAXH4VYRQYEX7S3XGK3H",
    "state": "TestState",
    "rangeBearing": {
      "elevation": 1.25,
      "azimuth": 180.5,
      "range": 450.0,
      "coordinateSystem": "LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M",
      "datum": "LOCATION_DATUM_WGS84_E"
    },
    "detectionConfidence": 0.91
  }
}`

const taskAckJSON = `{
  "timestamp": "2023-07-07T12:44:17.027638700Z",
  "nodeId": "a8654cdf-4328-47de-81fa-c495589e30c8",
  "taskAck": {
    "taskId": "01H4R63D7NVN8444Z5M77WEBY8",
    "taskStatus": "TASK_STATUS_ACCEPTED",
    "reason": ["accepted for preflight"]
  }
}`
