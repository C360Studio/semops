package semlink

import (
	"context"
	"fmt"
	"strings"
	"time"

	commandprojector "github.com/c360studio/semops/internal/projectors/command"
	mavlinkprojector "github.com/c360studio/semops/internal/projectors/mavlink"
)

const ClaimScopeCompanionIntentOnly = "semlink-companion-command-intent-only"

const (
	mavlinkCommandRequestMessage   = 512
	mavlinkMessageAutopilotVersion = 148
)

type ArduPilotReadbackRequest struct {
	CompanionNodeID    string
	ID                 string
	TargetAssetID      string
	TargetSystemID     int
	TargetComponentID  int
	CommandID          int
	RequestedMessageID int
	CorrelationID      string
	IdempotencyKey     string
	SourceRef          string
	RequestedAt        time.Time
	TTL                time.Duration
}

type Ingress struct {
	Projector       *commandprojector.GuardedProjector
	MAVLinkOrg      string
	MAVLinkPlatform string
	Clock           func() time.Time
}

type Result struct {
	ClaimScope               string
	Intent                   commandprojector.Intent
	Admission                commandprojector.AdmissionResult
	NativeExecutionAllowed   bool
	CompanionTransmitAllowed bool
}

func (i Ingress) AdmitArduPilotReadback(
	ctx context.Context,
	req ArduPilotReadbackRequest,
) (Result, commandprojector.Plan, error) {
	if i.Projector == nil {
		return Result{}, commandprojector.Plan{}, fmt.Errorf("semlink companion ingress requires guarded command projector")
	}
	if err := validateMAVLinkReadback(req); err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	targetAssetID, err := i.targetAssetID(req)
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	intent, err := commandprojector.NewSemLinkArduPilotReadbackIntent(commandprojector.SemLinkCompanionRequest{
		MeshNodeID:       req.CompanionNodeID,
		NativeID:         nativeID(req),
		TargetAssetID:    targetAssetID,
		VehicleSystemID:  req.TargetSystemID,
		VehicleComponent: req.TargetComponentID,
		CorrelationID:    req.CorrelationID,
		IdempotencyKey:   req.IdempotencyKey,
		SourceRef:        sourceRef(req),
		ObservedAt:       normalizeTime(req.RequestedAt, i.now()),
		TTL:              req.TTL,
	})
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	admission, plan, err := i.Projector.ProjectIntent(ctx, intent)
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	return Result{
		ClaimScope:               ClaimScopeCompanionIntentOnly,
		Intent:                   intent,
		Admission:                admission,
		NativeExecutionAllowed:   false,
		CompanionTransmitAllowed: false,
	}, plan, nil
}

func (i Ingress) targetAssetID(req ArduPilotReadbackRequest) (string, error) {
	if trimmed := strings.TrimSpace(req.TargetAssetID); trimmed != "" {
		return trimmed, nil
	}
	if req.TargetSystemID < 1 || req.TargetSystemID > 255 {
		return "", fmt.Errorf("SemLink companion target_system_id must be between 1 and 255, got %d", req.TargetSystemID)
	}
	return mavlinkprojector.SourceAssetID(i.MAVLinkOrg, i.MAVLinkPlatform, req.TargetSystemID), nil
}

func (i Ingress) now() time.Time {
	if i.Clock != nil {
		return i.Clock().UTC()
	}
	return time.Now().UTC()
}

func nativeID(req ArduPilotReadbackRequest) string {
	if trimmed := strings.TrimSpace(req.ID); trimmed != "" {
		return "semlink-" + safeToken(trimmed)
	}
	return "semlink-" + safeToken(req.CompanionNodeID) + "-autopilot-version"
}

func sourceRef(req ArduPilotReadbackRequest) string {
	if trimmed := strings.TrimSpace(req.SourceRef); trimmed != "" {
		return trimmed
	}
	return "semlink://" + safeToken(req.CompanionNodeID) + "/ardupilot/system-" + safeToken(fmt.Sprint(req.TargetSystemID)) + "/request-autopilot-version"
}

func validateMAVLinkReadback(req ArduPilotReadbackRequest) error {
	commandID := req.CommandID
	if commandID == 0 {
		commandID = mavlinkCommandRequestMessage
	}
	if commandID != mavlinkCommandRequestMessage {
		return fmt.Errorf("unsupported command_id %d; MVP allowlist: MAV_CMD_REQUEST_MESSAGE %d", commandID, mavlinkCommandRequestMessage)
	}
	requestedMessageID := req.RequestedMessageID
	if requestedMessageID == 0 {
		requestedMessageID = mavlinkMessageAutopilotVersion
	}
	if requestedMessageID != mavlinkMessageAutopilotVersion {
		return fmt.Errorf("unsupported requested_message_id %d; MVP allowlist: AUTOPILOT_VERSION %d", requestedMessageID, mavlinkMessageAutopilotVersion)
	}
	return nil
}

func normalizeTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback.UTC()
	}
	return value.UTC()
}

func safeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "unknown"
	}
	return token
}
