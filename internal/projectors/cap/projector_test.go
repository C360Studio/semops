package cap

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestProjectorCreatesAppendEvidenceHazard(t *testing.T) {
	alert := testAlert(t)
	projector := NewProjector(Config{
		Org:         "c360",
		Platform:    "edge",
		OwnerTokens: testOwnerTokens("test"),
		TraceID:     "scenario-cap-001",
	})

	plan, err := projector.ProjectAlert(alert, "cap://fixture/nws-demo-flood-warning")
	if err != nil {
		t.Fatalf("project alert: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want hazard evidence birth", len(plan.Mutations))
	}
	create := requireCreate(t, plan.Mutations[0])

	if create.Entity.ID != "c360.edge.cop.cap.hazard_area.nws-demo-flood-warning" {
		t.Fatalf("hazard id = %q", create.Entity.ID)
	}
	if create.Entity.MessageType.Key() != cop.CAPHazardEvidenceContract().MessageType {
		t.Fatalf("message type = %q", create.Entity.MessageType.Key())
	}
	if create.IndexingProfile != cop.CAPHazardEvidenceContract().IndexingProfile {
		t.Fatalf("indexing profile = %q", create.IndexingProfile)
	}
	if create.OwnerToken != "semops.feed.cap#test" {
		t.Fatalf("owner token = %q", create.OwnerToken)
	}
	if create.TraceID != "scenario-cap-001" {
		t.Fatalf("trace id = %q", create.TraceID)
	}
	requireTriple(t, create.Triples, cop.HazardSource, "cap")
	requireTriple(t, create.Triples, cop.ProvenanceSource, "cap")
	requireTriple(t, create.Triples, cop.ProvenanceSourceRef, "cap://fixture/nws-demo-flood-warning")
	if hasPredicate(create.Triples, cop.HazardGeometry) ||
		hasPredicate(create.Triples, cop.HazardSeverity) ||
		hasPredicate(create.Triples, cop.HazardStatus) {
		t.Fatalf("CAP evidence must not own authoritative hazard predicates: %+v", create.Triples)
	}

	advisory := stringTriple(t, create.Triples, cop.HazardAdvisoryText)
	if !strings.Contains(advisory, "Flood Warning issued") || !strings.Contains(advisory, "Move to higher ground") {
		t.Fatalf("advisory text = %q", advisory)
	}
	var evidence cop.HazardEvidenceDocument
	if err := json.Unmarshal([]byte(stringTriple(t, create.Triples, cop.HazardEvidence)), &evidence); err != nil {
		t.Fatalf("decode hazard evidence: %v", err)
	}
	if evidence.Identifier != alert.Identifier ||
		evidence.Event != "Flood Warning" ||
		evidence.Severity != "Severe" ||
		len(evidence.Polygons) != 1 ||
		len(evidence.Circles) != 1 {
		t.Fatalf("evidence = %+v", evidence)
	}
	if evidence.Polygons[0][0] != (cop.HazardEvidencePoint{Lat: 38.895, Lon: -77.012}) {
		t.Fatalf("first evidence point = %+v", evidence.Polygons[0][0])
	}
}

func TestProjectorAppendsForKnownHazard(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	alert := testAlert(t)

	birth, err := projector.ProjectAlert(alert, "cap://fixture/1")
	if err != nil {
		t.Fatalf("project birth: %v", err)
	}
	if marked := projector.MarkBornForPlan(birth); marked != 1 {
		t.Fatalf("marked births = %d, want 1", marked)
	}
	alert.MsgType = "Update"
	plan, err := projector.ProjectAlert(alert, "cap://fixture/2")
	if err != nil {
		t.Fatalf("project update: %v", err)
	}
	if len(plan.Mutations) != 1 {
		t.Fatalf("mutations = %d, want append update", len(plan.Mutations))
	}
	update := requireUpdate(t, plan.Mutations[0])
	if update.Entity.ID != "c360.edge.cop.cap.hazard_area.nws-demo-flood-warning" {
		t.Fatalf("update id = %q", update.Entity.ID)
	}
	if update.IndexingProfile != cop.CAPHazardEvidenceContract().IndexingProfile {
		t.Fatalf("update indexing profile = %q", update.IndexingProfile)
	}
	if update.OwnerToken != "semops.feed.cap#test" {
		t.Fatalf("update owner token = %q", update.OwnerToken)
	}
	var evidence cop.HazardEvidenceDocument
	if err := json.Unmarshal([]byte(stringTriple(t, update.AddTriples, cop.HazardEvidence)), &evidence); err != nil {
		t.Fatalf("decode update evidence: %v", err)
	}
	if evidence.MessageType != "Update" {
		t.Fatalf("message type = %q", evidence.MessageType)
	}
}

func TestProjectorProjectsLifecycleReplayRecordsInBirthOrder(t *testing.T) {
	records, err := capcodec.LifecycleFixtureRecords(testTime())
	if err != nil {
		t.Fatalf("lifecycle fixtures: %v", err)
	}
	alerts := make([]SourceAlert, 0, len(records))
	for _, record := range records {
		alert, err := record.Alert()
		if err != nil {
			t.Fatalf("parse replay record %q: %v", record.Ref, err)
		}
		alerts = append(alerts, SourceAlert{Alert: alert, SourceRef: record.Ref})
	}
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test"), TraceID: "scenario-cap-lifecycle"})

	plan, err := projector.ProjectAlerts(alerts)
	if err != nil {
		t.Fatalf("project alerts: %v", err)
	}
	if len(plan.Mutations) != 4 {
		t.Fatalf("mutations = %d, want 4", len(plan.Mutations))
	}

	first := requireCreate(t, plan.Mutations[0])
	second := requireUpdate(t, plan.Mutations[1])
	third := requireUpdate(t, plan.Mutations[2])
	fourth := requireCreate(t, plan.Mutations[3])

	warningID := EntityID("c360", "edge", "nws-demo-flood-warning")
	expiredID := EntityID("c360", "edge", "nws-demo-flood-expired")
	if first.Entity.ID != warningID ||
		second.Entity.ID != warningID ||
		third.Entity.ID != warningID ||
		fourth.Entity.ID != expiredID {
		t.Fatalf("mutation ids = %q/%q/%q/%q", first.Entity.ID, second.Entity.ID, third.Entity.ID, fourth.Entity.ID)
	}
	requireTriple(t, first.Triples, cop.ProvenanceSourceRef, "cap://fixture/hadr-flood/0001-alert")
	requireTriple(t, second.AddTriples, cop.ProvenanceSourceRef, "cap://fixture/hadr-flood/0002-update")
	requireTriple(t, third.AddTriples, cop.ProvenanceSourceRef, "cap://fixture/hadr-flood/0003-cancel")
	requireTriple(t, fourth.Triples, cop.ProvenanceSourceRef, "cap://fixture/hadr-flood/0004-expired")

	var cancelEvidence cop.HazardEvidenceDocument
	if err := json.Unmarshal([]byte(stringTriple(t, third.AddTriples, cop.HazardEvidence)), &cancelEvidence); err != nil {
		t.Fatalf("decode cancel evidence: %v", err)
	}
	if cancelEvidence.MessageType != "Cancel" {
		t.Fatalf("cancel evidence message type = %q", cancelEvidence.MessageType)
	}
}

func TestProjectorRejectsInvalidAlert(t *testing.T) {
	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	_, err := projector.ProjectAlert(capcodec.Alert{}, "cap://bad")
	if err == nil {
		t.Fatal("expected invalid alert error")
	}
	if !strings.Contains(err.Error(), "identifier") {
		t.Fatalf("error = %v", err)
	}
}

func testAlert(t *testing.T) capcodec.Alert {
	t.Helper()
	alert, err := capcodec.Parse([]byte(sampleCAPAlert))
	if err != nil {
		t.Fatalf("parse test cap alert: %v", err)
	}
	return alert
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

func stringTriple(t *testing.T, triples []message.Triple, predicate string) string {
	t.Helper()
	for _, triple := range triples {
		if triple.Predicate == predicate {
			value, ok := triple.Object.(string)
			if !ok {
				t.Fatalf("%s object = %#v, want string", predicate, triple.Object)
			}
			return value
		}
	}
	t.Fatalf("missing predicate %q in %+v", predicate, triples)
	return ""
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
		cop.OwnerCAP: ownership.ExpectedOwnerToken(cop.OwnerCAP, incarnation),
	}
}

func testTime() time.Time {
	return time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
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
    <web>https://example.test/flood</web>
    <resource>
      <resourceDesc>Flood detail</resourceDesc>
      <mimeType>text/html</mimeType>
      <uri>https://example.test/flood-detail</uri>
    </resource>
    <area>
      <areaDesc>North Branch</areaDesc>
      <polygon>38.895,-77.012 38.907,-77.011 38.908,-76.992 38.896,-76.991</polygon>
      <circle>38.900,-77.010 7.5</circle>
    </area>
  </info>
</alert>`
