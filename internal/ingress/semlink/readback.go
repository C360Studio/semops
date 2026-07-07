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

type ArduPilotReadbackRequest struct {
	MeshNodeID       string
	ID               string
	TargetAssetID    string
	VehicleSystemID  int
	VehicleComponent int
	Action           string
	CorrelationID    string
	IdempotencyKey   string
	SourceRef        string
	ObservedAt       time.Time
	TTL              time.Duration
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
	targetAssetID, err := i.targetAssetID(req)
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	intent, err := commandprojector.NewSemLinkArduPilotReadbackIntent(commandprojector.SemLinkCompanionRequest{
		MeshNodeID:       req.MeshNodeID,
		NativeID:         nativeID(req),
		TargetAssetID:    targetAssetID,
		VehicleSystemID:  req.VehicleSystemID,
		VehicleComponent: req.VehicleComponent,
		Action:           req.Action,
		CorrelationID:    req.CorrelationID,
		IdempotencyKey:   req.IdempotencyKey,
		SourceRef:        sourceRef(req),
		ObservedAt:       normalizeTime(req.ObservedAt, i.now()),
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
	if req.VehicleSystemID < 1 || req.VehicleSystemID > 255 {
		return "", fmt.Errorf("SemLink companion vehicle_system_id must be between 1 and 255, got %d", req.VehicleSystemID)
	}
	return mavlinkprojector.SourceAssetID(i.MAVLinkOrg, i.MAVLinkPlatform, req.VehicleSystemID), nil
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
	return "semlink-" + safeToken(req.MeshNodeID) + "-autopilot-version"
}

func sourceRef(req ArduPilotReadbackRequest) string {
	if trimmed := strings.TrimSpace(req.SourceRef); trimmed != "" {
		return trimmed
	}
	return "semlink://" + safeToken(req.MeshNodeID) + "/ardupilot/system-" + safeToken(fmt.Sprint(req.VehicleSystemID)) + "/request-autopilot-version"
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
