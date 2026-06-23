package command

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

type graphTriple = message.Triple

func TestProjectorCreatesCommandIntentWithoutBirthingTargetAsset(t *testing.T) {
	intent := sampleIntent()
	projector := NewProjector(Config{
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "command-intent-001",
	})
	plan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project command intent: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want command intent create only", len(plan.Mutations))
	}

	create := requireCreate(t, plan.Mutations[0])
	if create.Entity.ID != "c360.edge.cop.command.task.csapi-command-123" {
		t.Fatalf("command intent id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.CommandIntentContract().MessageType {
		t.Fatalf("message type = %q", create.Entity.MessageType.Key())
	}
	if create.Entity.UpdatedAt != intent.ObservedAt {
		t.Fatalf("updated_at = %s, want %s", create.Entity.UpdatedAt, intent.ObservedAt)
	}
	if create.IndexingProfile != cop.CommandIntentContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.OwnerToken != "semops.command.intent#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "command-intent-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}

	requireTriple(t, create.Triples, cop.TaskTarget, intent.TargetAssetID)
	requireTriple(t, create.Triples, cop.TaskNativeID, "csapi-command-123")
	requireTriple(t, create.Triples, cop.TaskName, "Route MAVLink system 42 to North Gate")
	requireTriple(t, create.Triples, cop.TaskKind, "mavlink.goto")
	requireTriple(t, create.Triples, cop.TaskStatus, "requested")
	requireTriple(t, create.Triples, cop.TaskDescription, "operator-approved route change")
	requireTriple(t, create.Triples, cop.TaskDesired, `{"command":"goto","lat":38.9,"lon":-77.04}`)
	requireTriple(t, create.Triples, cop.TaskAuthority, "local.operator")
	requireTriple(t, create.Triples, cop.TaskPriority, int64(80))
	requireTriple(t, create.Triples, cop.TaskExpiresAt, intent.ExpiresAt)
	requireTriple(t, create.Triples, cop.TaskCorrelation, "csapi:req-123")
	requireTriple(t, create.Triples, cop.TaskIdempotency, "idem-123")
	requireTriple(t, create.Triples, cop.TaskRequestedBy, "operator:coby")
	requireTriple(t, create.Triples, cop.ProvenanceSource, "cs-api")
	requireTriple(t, create.Triples, cop.ProvenanceConfidence, 1.0)
	requireTriple(t, create.Triples, cop.ProvenanceObservedAt, intent.ObservedAt)
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "csapi://commands/123")

	if hasEntityMessageType(plan, cop.SourceAssetContract().MessageType) {
		t.Fatal("command intent projector must not birth target source assets")
	}
}

func TestProjectorUpdatesKnownCommandIntentWithoutRepeatingTargetEdge(t *testing.T) {
	intent := sampleIntent()
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	birthPlan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project birth: %v", err)
	}
	if marked := projector.MarkBornForPlan(birthPlan); marked != 1 {
		t.Fatalf("marked births = %d, want command intent", marked)
	}

	intent.Status = "cancelled"
	intent.DesiredState = `{"command":"cancel","reason":"local_override"}`
	updatePlan, err := projector.ProjectIntent(intent)
	if err != nil {
		t.Fatalf("project update: %v", err)
	}
	if len(updatePlan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want update", len(updatePlan.Mutations))
	}
	update := requireUpdate(t, updatePlan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.command.task.csapi-command-123" {
		t.Fatalf("update id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.CommandIntentContract().IndexingProfile {
		t.Fatalf("update indexing profile = %q", update.IndexingProfile)
	}
	if hasPredicate(update.AddTriples, cop.TaskTarget) {
		t.Fatal("known command intent updates must not repeat strict target edge")
	}
	requireTriple(t, update.AddTriples, cop.TaskStatus, "cancelled")
	requireTriple(t, update.AddTriples, cop.TaskDesired, `{"command":"cancel","reason":"local_override"}`)
}

func TestProjectorRejectsMalformedCommandIntent(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Intent)
		want string
	}{
		{name: "missing native id", edit: func(i *Intent) { i.NativeID = "" }, want: "native_id"},
		{name: "missing target", edit: func(i *Intent) { i.TargetAssetID = "" }, want: "target asset id"},
		{name: "missing kind", edit: func(i *Intent) { i.Kind = "" }, want: "kind"},
		{name: "missing desired state", edit: func(i *Intent) { i.DesiredState = "" }, want: "desired_state"},
		{name: "missing authority", edit: func(i *Intent) { i.Authority = "" }, want: "authority"},
		{name: "priority too low", edit: func(i *Intent) { i.Priority = 0 }, want: "priority"},
		{name: "priority too high", edit: func(i *Intent) { i.Priority = 101 }, want: "priority"},
		{name: "missing expiry", edit: func(i *Intent) { i.ExpiresAt = time.Time{} }, want: "expires_at"},
		{name: "expired before observation", edit: func(i *Intent) { i.ExpiresAt = i.ObservedAt }, want: "after observed_at"},
		{name: "missing correlation", edit: func(i *Intent) { i.CorrelationID = "" }, want: "correlation_id"},
		{name: "missing idempotency", edit: func(i *Intent) { i.IdempotencyKey = "" }, want: "idempotency_key"},
		{name: "missing requested by", edit: func(i *Intent) { i.RequestedBy = "" }, want: "requested_by"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := sampleIntent()
			tt.edit(&intent)
			_, err := NewProjector(Config{}).ProjectIntent(intent)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func sampleIntent() Intent {
	observedAt := time.Date(2026, 6, 23, 20, 0, 0, 0, time.UTC)
	return Intent{
		NativeID:       "csapi-command-123",
		TargetAssetID:  "c360.edge.cop.mavlink.asset.system-42",
		Name:           "Route MAVLink system 42 to North Gate",
		Kind:           "mavlink.goto",
		Status:         "requested",
		Description:    "operator-approved route change",
		DesiredState:   `{"command":"goto","lat":38.9,"lon":-77.04}`,
		Authority:      "local.operator",
		Priority:       80,
		ExpiresAt:      observedAt.Add(2 * time.Minute),
		CorrelationID:  "csapi:req-123",
		IdempotencyKey: "idem-123",
		RequestedBy:    "operator:coby",
		ObservedAt:     observedAt,
		Source:         "cs-api",
		SourceRef:      "csapi://commands/123",
	}
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

func requireTriple(t *testing.T, triples []graphTriple, predicate string, want any) {
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

func hasPredicate(triples []graphTriple, predicate string) bool {
	for _, triple := range triples {
		if triple.Predicate == predicate {
			return true
		}
	}
	return false
}

func hasEntityMessageType(plan Plan, messageType string) bool {
	for _, mutation := range plan.Mutations {
		if mutation.Kind == MutationCreate &&
			mutation.Create.Entity != nil &&
			mutation.Create.Entity.MessageType.Key() == messageType {
			return true
		}
	}
	return false
}

func testOwnerTokens(incarnation string) map[string]ownership.OwnerToken {
	return map[string]ownership.OwnerToken{
		cop.OwnerCommand: ownership.ExpectedOwnerToken(cop.OwnerCommand, incarnation),
	}
}
