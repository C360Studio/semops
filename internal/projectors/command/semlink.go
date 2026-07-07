package command

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	SemLinkActionRequestAutopilotVersion = "request_autopilot_version"
	SemLinkCompanionAuthority            = "semlink.companion"
	SemLinkCompanionSource               = "semlink.companion.mesh"
)

const (
	mavlinkCommandRequestMessage   = 512
	mavlinkMessageAutopilotVersion = 148
)

type SemLinkCompanionRequest struct {
	MeshNodeID       string
	NativeID         string
	TargetAssetID    string
	VehicleSystemID  int
	VehicleComponent int
	Action           string
	CorrelationID    string
	IdempotencyKey   string
	SourceRef        string
	ObservedAt       time.Time
	ExpiresAt        time.Time
	TTL              time.Duration
	Priority         int
}

type semLinkDesiredState struct {
	Command            string `json:"command"`
	Message            string `json:"message"`
	MAVLinkCommand     int    `json:"mavlink_command"`
	MessageID          int    `json:"message_id"`
	VehicleSystemID    int    `json:"vehicle_system_id"`
	VehicleComponentID int    `json:"vehicle_component_id"`
}

func NewSemLinkArduPilotReadbackIntent(req SemLinkCompanionRequest) (Intent, error) {
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		req.Action = SemLinkActionRequestAutopilotVersion
	}
	if req.Action != SemLinkActionRequestAutopilotVersion {
		return Intent{}, fmt.Errorf("unsupported SemLink ArduPilot action %q; MVP allowlist: %s", req.Action, SemLinkActionRequestAutopilotVersion)
	}
	if strings.TrimSpace(req.MeshNodeID) == "" {
		return Intent{}, fmt.Errorf("SemLink companion mesh_node_id is required")
	}
	if strings.TrimSpace(req.NativeID) == "" {
		return Intent{}, fmt.Errorf("SemLink companion native_id is required")
	}
	if strings.TrimSpace(req.TargetAssetID) == "" {
		return Intent{}, fmt.Errorf("SemLink companion target_asset_id is required")
	}
	if req.VehicleSystemID < 1 || req.VehicleSystemID > 255 {
		return Intent{}, fmt.Errorf("SemLink companion vehicle_system_id must be between 1 and 255, got %d", req.VehicleSystemID)
	}
	if req.VehicleComponent == 0 {
		req.VehicleComponent = 1
	}
	if req.VehicleComponent < 1 || req.VehicleComponent > 255 {
		return Intent{}, fmt.Errorf("SemLink companion vehicle_component_id must be between 1 and 255, got %d", req.VehicleComponent)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		return Intent{}, fmt.Errorf("SemLink companion correlation_id is required")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return Intent{}, fmt.Errorf("SemLink companion idempotency_key is required")
	}

	observedAt := req.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	expiresAt := req.ExpiresAt.UTC()
	if expiresAt.IsZero() {
		ttl := req.TTL
		if ttl == 0 {
			ttl = 30 * time.Second
		}
		if ttl < 0 {
			return Intent{}, fmt.Errorf("SemLink companion ttl must be positive, got %s", ttl)
		}
		expiresAt = observedAt.Add(ttl)
	}
	if !expiresAt.After(observedAt) {
		return Intent{}, fmt.Errorf("SemLink companion expires_at must be after observed_at")
	}

	priority := req.Priority
	if priority == 0 {
		priority = 50
	}

	desired, err := json.Marshal(semLinkDesiredState{
		Command:            "MAV_CMD_REQUEST_MESSAGE",
		Message:            "AUTOPILOT_VERSION",
		MAVLinkCommand:     mavlinkCommandRequestMessage,
		MessageID:          mavlinkMessageAutopilotVersion,
		VehicleSystemID:    req.VehicleSystemID,
		VehicleComponentID: req.VehicleComponent,
	})
	if err != nil {
		return Intent{}, fmt.Errorf("marshal SemLink companion desired state: %w", err)
	}

	meshNodeID := strings.TrimSpace(req.MeshNodeID)
	return Intent{
		NativeID:            strings.TrimSpace(req.NativeID),
		TargetAssetID:       strings.TrimSpace(req.TargetAssetID),
		Name:                "Request ArduPilot AUTOPILOT_VERSION from " + meshNodeID,
		Kind:                "mavlink.request_message",
		Status:              StatusRequested,
		Description:         "SemLink companion ArduPilot readback request",
		DesiredState:        string(desired),
		Authority:           SemLinkCompanionAuthority,
		Priority:            priority,
		ExpiresAt:           expiresAt,
		CorrelationID:       strings.TrimSpace(req.CorrelationID),
		IdempotencyKey:      strings.TrimSpace(req.IdempotencyKey),
		RequestedBy:         "semlink:" + meshNodeID,
		LocalOverridePolicy: LocalOverrideNotRequired,
		ObservedAt:          observedAt,
		Source:              SemLinkCompanionSource,
		SourceRef:           strings.TrimSpace(req.SourceRef),
	}, nil
}
