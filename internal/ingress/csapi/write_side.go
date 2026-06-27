package csapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	commandprojector "github.com/c360studio/semops/internal/projectors/command"
)

type Surface string

const (
	SurfaceCommand       Surface = "Command"
	SurfaceControlStream Surface = "ControlStream"

	ClaimScopeCommandIntentOnly = "write-side-command-intent-only"
	SourceCSAPI                 = "cs-api"
)

type CommandRequest struct {
	ID                     string
	TargetAssetID          string
	Name                   string
	Kind                   string
	Description            string
	DesiredState           string
	Authority              string
	AuthorityAuthenticated bool
	Priority               int
	TTL                    time.Duration
	Deadline               time.Time
	CorrelationID          string
	IdempotencyKey         string
	RequestedBy            string
	LocalOverridePolicy    string
	ObservedAt             time.Time
	SourceRef              string
}

type ControlStreamRequest struct {
	StreamID string
	Command  CommandRequest
}

type Ingress struct {
	Projector *commandprojector.GuardedProjector
	Clock     func() time.Time
}

type Result struct {
	Surface                          Surface
	ClaimScope                       string
	Intent                           commandprojector.Intent
	Admission                        commandprojector.AdmissionResult
	NativeExecutionAllowed           bool
	UpstreamStatusPublicationAllowed bool
}

func (i Ingress) AdmitCommand(ctx context.Context, req CommandRequest) (Result, commandprojector.Plan, error) {
	intent, err := IntentFromCommand(i.now(), req)
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	return i.admit(ctx, SurfaceCommand, intent)
}

func (i Ingress) AdmitControlStream(ctx context.Context, req ControlStreamRequest) (Result, commandprojector.Plan, error) {
	intent, err := IntentFromControlStream(i.now(), req)
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	return i.admit(ctx, SurfaceControlStream, intent)
}

func IntentFromCommand(now time.Time, req CommandRequest) (commandprojector.Intent, error) {
	return intentFromRequest(now, SurfaceCommand, "", req)
}

func IntentFromControlStream(now time.Time, req ControlStreamRequest) (commandprojector.Intent, error) {
	streamID := strings.TrimSpace(req.StreamID)
	if streamID == "" {
		return commandprojector.Intent{}, fmt.Errorf("cs api controlstream id is required")
	}
	return intentFromRequest(now, SurfaceControlStream, streamID, req.Command)
}

func (i Ingress) admit(ctx context.Context, surface Surface, intent commandprojector.Intent) (Result, commandprojector.Plan, error) {
	if i.Projector == nil {
		return Result{}, commandprojector.Plan{}, fmt.Errorf("cs api write-side ingress requires guarded command projector")
	}
	admission, plan, err := i.Projector.ProjectIntent(ctx, intent)
	if err != nil {
		return Result{}, commandprojector.Plan{}, err
	}
	return Result{
		Surface:                          surface,
		ClaimScope:                       ClaimScopeCommandIntentOnly,
		Intent:                           intent,
		Admission:                        admission,
		NativeExecutionAllowed:           false,
		UpstreamStatusPublicationAllowed: false,
	}, plan, nil
}

func (i Ingress) now() time.Time {
	if i.Clock != nil {
		return i.Clock().UTC()
	}
	return time.Now().UTC()
}

func intentFromRequest(now time.Time, surface Surface, streamID string, req CommandRequest) (commandprojector.Intent, error) {
	now = normalizeTime(now, time.Now().UTC())
	observedAt := normalizeTime(req.ObservedAt, now)
	expiresAt, err := deadline(req, observedAt)
	if err != nil {
		return commandprojector.Intent{}, err
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		return commandprojector.Intent{}, fmt.Errorf("cs api command id is required")
	}
	if !req.AuthorityAuthenticated {
		return commandprojector.Intent{}, fmt.Errorf("cs api command authority must be authenticated")
	}
	kind := firstNonEmpty(req.Kind, defaultKind(surface))
	return commandprojector.Intent{
		NativeID:            nativeID(surface, streamID, id),
		TargetAssetID:       strings.TrimSpace(req.TargetAssetID),
		Name:                firstNonEmpty(req.Name, kind, id),
		Kind:                kind,
		Status:              commandprojector.StatusRequested,
		Description:         strings.TrimSpace(req.Description),
		DesiredState:        strings.TrimSpace(req.DesiredState),
		Authority:           strings.TrimSpace(req.Authority),
		Priority:            req.Priority,
		ExpiresAt:           expiresAt,
		CorrelationID:       strings.TrimSpace(req.CorrelationID),
		IdempotencyKey:      strings.TrimSpace(req.IdempotencyKey),
		RequestedBy:         strings.TrimSpace(req.RequestedBy),
		LocalOverridePolicy: strings.TrimSpace(req.LocalOverridePolicy),
		ObservedAt:          observedAt,
		Source:              SourceCSAPI,
		SourceRef:           sourceRef(surface, streamID, id, req.SourceRef),
	}, nil
}

func deadline(req CommandRequest, observedAt time.Time) (time.Time, error) {
	if !req.Deadline.IsZero() {
		return req.Deadline.UTC(), nil
	}
	if req.TTL <= 0 {
		return time.Time{}, fmt.Errorf("cs api command ttl or deadline is required")
	}
	return observedAt.Add(req.TTL).UTC(), nil
}

func sourceRef(surface Surface, streamID string, id string, explicit string) string {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return trimmed
	}
	switch surface {
	case SurfaceControlStream:
		return "csapi://controlstreams/" + safeToken(streamID) + "/commands/" + safeToken(id)
	default:
		return "csapi://commands/" + safeToken(id)
	}
}

func nativeID(surface Surface, streamID string, id string) string {
	switch surface {
	case SurfaceControlStream:
		return "csapi-controlstream-" + safeToken(streamID) + "-" + safeToken(id)
	default:
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(id)), "csapi-command-") {
			return strings.TrimSpace(id)
		}
		return "csapi-command-" + safeToken(id)
	}
}

func defaultKind(surface Surface) string {
	switch surface {
	case SurfaceControlStream:
		return "csapi.controlstream.command"
	default:
		return "csapi.command"
	}
}

func normalizeTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback.UTC()
	}
	return value.UTC()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
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
		case !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "unknown"
	}
	return token
}
