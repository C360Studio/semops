package csapi

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type semLinkReadbackRequestFixture struct {
	Contract           string    `json:"contract"`
	CompanionNodeID    string    `json:"companion_node_id"`
	TargetSystemID     int       `json:"target_system_id"`
	TargetComponentID  int       `json:"target_component_id"`
	CommandID          int       `json:"command_id"`
	RequestedMessageID int       `json:"requested_message_id"`
	CorrelationID      string    `json:"correlation_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	RequestedAt        time.Time `json:"requested_at"`
	TTLSeconds         int       `json:"ttl_seconds"`
	SourceRef          string    `json:"source_ref"`
}

type semLinkReadbackResponseFixture struct {
	Contract                  string    `json:"contract"`
	Accepted                  bool      `json:"accepted"`
	Status                    string    `json:"status"`
	Duplicate                 bool      `json:"duplicate"`
	CorrelationID             string    `json:"correlation_id"`
	IdempotencyKey            string    `json:"idempotency_key"`
	CompanionNodeID           string    `json:"companion_node_id"`
	AuthorizedCompanionNodeID string    `json:"authorized_companion_node_id"`
	AuthorityScope            string    `json:"authority_scope"`
	TargetAssetID             string    `json:"target_asset_id"`
	EntityID                  string    `json:"entity_id"`
	NativeID                  string    `json:"native_id"`
	ClaimScope                string    `json:"claim_scope"`
	SourceRef                 string    `json:"source_ref"`
	RequestedAt               time.Time `json:"requested_at"`
	ExpiresAt                 time.Time `json:"expires_at"`
	NativeExecutionAllowed    bool      `json:"native_execution_allowed"`
	CompanionTransmitAllowed  bool      `json:"companion_transmit_allowed"`
	Mutations                 int       `json:"mutations"`
}

type semLinkCSAPIProjectionFixture struct {
	Contract         string                      `json:"contract"`
	SourceContract   string                      `json:"source_contract"`
	ClaimScope       string                      `json:"claim_scope"`
	CSAPIScore       string                      `json:"csapi_score"`
	Systems          []projectionSystem          `json:"systems"`
	ControlStream    projectionControlStream     `json:"control_stream"`
	Command          projectionCommand           `json:"command"`
	SystemEvents     []projectionSystemEvent     `json:"system_events"`
	Observations     []projectionObservation     `json:"observations"`
	DeferredSurfaces []projectionDeferredSurface `json:"deferred_surfaces"`
}

type projectionSystem struct {
	Role      string        `json:"role"`
	ID        string        `json:"id"`
	MapsTo    string        `json:"maps_to"`
	SourceRef string        `json:"source_ref"`
	MAVLink   mavlinkFields `json:"mavlink"`
}

type projectionControlStream struct {
	ID                string           `json:"id"`
	SystemID          string           `json:"system_id"`
	CompanionSystemID string           `json:"companion_system_id"`
	MapsTo            string           `json:"maps_to"`
	MAVLink           mavlinkFields    `json:"mavlink"`
	Governance        governanceFields `json:"governance"`
}

type projectionCommand struct {
	ID                string           `json:"id"`
	NativeID          string           `json:"native_id"`
	ControlStreamID   string           `json:"control_stream_id"`
	SystemID          string           `json:"system_id"`
	CompanionSystemID string           `json:"companion_system_id"`
	MapsTo            string           `json:"maps_to"`
	Status            string           `json:"status"`
	CorrelationID     string           `json:"correlation_id"`
	IdempotencyKey    string           `json:"idempotency_key"`
	SourceRef         string           `json:"source_ref"`
	IssueTime         time.Time        `json:"issue_time"`
	ExpiresAt         time.Time        `json:"expires_at"`
	Desired           mavlinkFields    `json:"desired"`
	Governance        governanceFields `json:"governance"`
}

type projectionSystemEvent struct {
	ID         string         `json:"id"`
	SystemID   string         `json:"system_id"`
	EventType  string         `json:"event_type"`
	MapsTo     string         `json:"maps_to"`
	Status     string         `json:"status"`
	ObservedAt time.Time      `json:"observed_at"`
	Payload    map[string]any `json:"payload"`
}

type projectionObservation struct {
	ID               string `json:"id"`
	SystemID         string `json:"system_id"`
	ObservedProperty string `json:"observed_property"`
	MapsTo           string `json:"maps_to"`
	Deferred         bool   `json:"deferred"`
	Reason           string `json:"reason"`
}

type projectionDeferredSurface struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type mavlinkFields struct {
	Command            string `json:"command"`
	Message            string `json:"message"`
	CommandID          int    `json:"command_id"`
	RequestedMessageID int    `json:"requested_message_id"`
	TargetSystemID     int    `json:"target_system_id"`
	TargetComponentID  int    `json:"target_component_id"`
}

type governanceFields struct {
	AuthorityScope           string `json:"authority_scope"`
	ClaimScope               string `json:"claim_scope"`
	NativeExecutionAllowed   bool   `json:"native_execution_allowed"`
	CompanionTransmitAllowed bool   `json:"companion_transmit_allowed"`
	Duplicate                bool   `json:"duplicate"`
	Mutations                int    `json:"mutations"`
}

func TestSemLinkReadbackCSAPIProjectionFixturePreservesContractFields(t *testing.T) {
	request := readContractFixture[semLinkReadbackRequestFixture](t, "request.accepted.json")
	response := readContractFixture[semLinkReadbackResponseFixture](t, "response.accepted.json")
	projection := readContractFixture[semLinkCSAPIProjectionFixture](t, "csapi-projection.accepted.json")

	if projection.SourceContract != request.Contract || response.Contract != request.Contract {
		t.Fatalf("contract chain = request %q response %q projection source %q", request.Contract, response.Contract, projection.SourceContract)
	}
	if projection.CSAPIScore != "amber" || projection.ClaimScope != "csapi-projection-fixture-only" {
		t.Fatalf("projection posture = score %q claim %q", projection.CSAPIScore, projection.ClaimScope)
	}

	companion, ok := projectionSystemByRole(projection, "companion")
	if !ok || companion.ID == "" || companion.MapsTo != "CS API System" {
		t.Fatalf("companion system = %+v", companion)
	}
	target, ok := projectionSystemByRole(projection, "target")
	if !ok || target.ID != response.TargetAssetID || target.MapsTo != "CS API System" {
		t.Fatalf("target system = %+v, response target %q", target, response.TargetAssetID)
	}
	requireMAVLinkTarget(t, target.MAVLink, request)

	stream := projection.ControlStream
	if stream.MapsTo != "CS API ControlStream" ||
		stream.SystemID != target.ID ||
		stream.CompanionSystemID != companion.ID {
		t.Fatalf("control stream = %+v", stream)
	}
	requireMAVLinkRequest(t, stream.MAVLink, request)
	if stream.Governance.NativeExecutionAllowed || stream.Governance.CompanionTransmitAllowed {
		t.Fatalf("control stream grants transmit authority: %+v", stream.Governance)
	}

	command := projection.Command
	if command.MapsTo != "CS API Command" ||
		command.ID != response.EntityID ||
		command.NativeID != response.NativeID ||
		command.ControlStreamID != stream.ID ||
		command.SystemID != target.ID ||
		command.CompanionSystemID != companion.ID ||
		command.Status != response.Status {
		t.Fatalf("command projection = %+v", command)
	}
	if command.CorrelationID != request.CorrelationID ||
		command.IdempotencyKey != request.IdempotencyKey ||
		command.SourceRef != request.SourceRef ||
		command.IssueTime != request.RequestedAt ||
		command.ExpiresAt != response.ExpiresAt {
		t.Fatalf("command trace/timing = %+v", command)
	}
	requireMAVLinkRequest(t, command.Desired, request)
	if command.Governance.AuthorityScope != response.AuthorityScope ||
		command.Governance.ClaimScope != response.ClaimScope ||
		command.Governance.Duplicate != response.Duplicate ||
		command.Governance.Mutations != response.Mutations ||
		command.Governance.NativeExecutionAllowed != response.NativeExecutionAllowed ||
		command.Governance.CompanionTransmitAllowed != response.CompanionTransmitAllowed {
		t.Fatalf("command governance = %+v, response = %+v", command.Governance, response)
	}

	if len(projection.SystemEvents) != 1 {
		t.Fatalf("system events = %+v", projection.SystemEvents)
	}
	event := projection.SystemEvents[0]
	if event.MapsTo != "CS API SystemEvent" ||
		event.SystemID != target.ID ||
		event.Status != response.Status ||
		event.ObservedAt != request.RequestedAt ||
		event.Payload["correlation_id"] != request.CorrelationID ||
		event.Payload["idempotency_key"] != request.IdempotencyKey ||
		event.Payload["native_execution_allowed"] != false ||
		event.Payload["companion_transmit_allowed"] != false {
		t.Fatalf("system event = %+v", event)
	}

	if len(projection.Observations) != 1 ||
		projection.Observations[0].ObservedProperty != "AUTOPILOT_VERSION" ||
		!projection.Observations[0].Deferred {
		t.Fatalf("AUTOPILOT_VERSION observation must remain deferred until structured payload exists: %+v", projection.Observations)
	}
	for _, name := range []string{"csapi.command-status-egress", "csapi.autopilot-version-observation", "csapi.raw-mavlink-transport"} {
		if !hasProjectionDeferredSurface(projection, name) {
			t.Fatalf("missing deferred surface %q in %+v", name, projection.DeferredSurfaces)
		}
	}
}

func readContractFixture[T any](t *testing.T, name string) T {
	t.Helper()
	body, err := os.ReadFile("../../../testdata/contracts/semlink-companion-readback-v0/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return out
}

func projectionSystemByRole(projection semLinkCSAPIProjectionFixture, role string) (projectionSystem, bool) {
	for _, system := range projection.Systems {
		if system.Role == role {
			return system, true
		}
	}
	return projectionSystem{}, false
}

func requireMAVLinkTarget(t *testing.T, fields mavlinkFields, request semLinkReadbackRequestFixture) {
	t.Helper()
	if fields.TargetSystemID != request.TargetSystemID ||
		fields.TargetComponentID != request.TargetComponentID {
		t.Fatalf("MAVLink target = %+v, request = %+v", fields, request)
	}
}

func requireMAVLinkRequest(t *testing.T, fields mavlinkFields, request semLinkReadbackRequestFixture) {
	t.Helper()
	requireMAVLinkTarget(t, fields, request)
	if fields.CommandID != request.CommandID ||
		fields.RequestedMessageID != request.RequestedMessageID {
		t.Fatalf("MAVLink request = %+v, request = %+v", fields, request)
	}
}

func hasProjectionDeferredSurface(projection semLinkCSAPIProjectionFixture, name string) bool {
	for _, surface := range projection.DeferredSurfaces {
		if surface.Name == name {
			return true
		}
	}
	return false
}
