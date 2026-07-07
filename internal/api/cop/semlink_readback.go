package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	semlinkingress "github.com/c360studio/semops/internal/ingress/semlink"
	commandprojector "github.com/c360studio/semops/internal/projectors/command"
)

const (
	semLinkReadbackMaxBodyBytes        = 8192
	semLinkReadbackContractV0          = "c360.semops.semlink.ardupilot.readback.v0"
	semLinkReadbackCommandRequestMsg   = 512
	semLinkReadbackMessageAutopilotVer = 148
)

type SemLinkReadbackIngress interface {
	AdmitArduPilotReadback(
		context.Context,
		semlinkingress.ArduPilotReadbackRequest,
	) (semlinkingress.Result, commandprojector.Plan, error)
}

type CommandPlanWriter interface {
	Apply(context.Context, commandprojector.Plan) error
}

type semLinkArduPilotReadbackRequest struct {
	Contract           string    `json:"contract,omitempty"`
	CompanionNodeID    string    `json:"companion_node_id,omitempty"`
	MeshNodeID         string    `json:"mesh_node_id,omitempty"`
	ID                 string    `json:"id,omitempty"`
	TargetAssetID      string    `json:"target_asset_id,omitempty"`
	TargetSystemID     int       `json:"target_system_id,omitempty"`
	TargetComponentID  int       `json:"target_component_id,omitempty"`
	CommandID          int       `json:"command_id,omitempty"`
	RequestedMessageID int       `json:"requested_message_id,omitempty"`
	VehicleSystemID    int       `json:"vehicle_system_id,omitempty"`
	VehicleComponentID int       `json:"vehicle_component_id,omitempty"`
	Action             string    `json:"action,omitempty"`
	CorrelationID      string    `json:"correlation_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	SourceRef          string    `json:"source_ref,omitempty"`
	RequestedAt        time.Time `json:"requested_at,omitempty"`
	ObservedAt         time.Time `json:"observed_at,omitempty"`
	TTLSeconds         int       `json:"ttl_seconds,omitempty"`
}

type semLinkReadbackResponse struct {
	Contract                  string    `json:"contract,omitempty"`
	Accepted                  bool      `json:"accepted"`
	Status                    string    `json:"status,omitempty"`
	RejectedReason            string    `json:"rejected_reason,omitempty"`
	Duplicate                 bool      `json:"duplicate"`
	ExistingNativeID          string    `json:"existing_native_id,omitempty"`
	ClaimScope                string    `json:"claim_scope,omitempty"`
	EntityID                  string    `json:"entity_id,omitempty"`
	NativeID                  string    `json:"native_id,omitempty"`
	TargetAssetID             string    `json:"target_asset_id,omitempty"`
	SourceRef                 string    `json:"source_ref,omitempty"`
	CorrelationID             string    `json:"correlation_id,omitempty"`
	IdempotencyKey            string    `json:"idempotency_key,omitempty"`
	CompanionNodeID           string    `json:"companion_node_id,omitempty"`
	RequestedAt               time.Time `json:"requested_at,omitempty"`
	ExpiresAt                 time.Time `json:"expires_at,omitempty"`
	Mutations                 int       `json:"mutations"`
	NativeExecutionAllowed    bool      `json:"native_execution_allowed"`
	CompanionTransmitAllowed  bool      `json:"companion_transmit_allowed"`
	AuthorizedBy              string    `json:"authorized_by,omitempty"`
	AuthorizedCompanionNodeID string    `json:"authorized_companion_node_id,omitempty"`
	AuthorizedMeshNodeID      string    `json:"authorized_mesh_node_id,omitempty"`
	AuthorityScope            string    `json:"authority_scope,omitempty"`
	AuthorityDomain           string    `json:"authority_domain,omitempty"`
	Authenticated             bool      `json:"authenticated,omitempty"`
	Error                     string    `json:"error,omitempty"`
}

func (h *Handler) admitSemLinkArduPilotReadback(w http.ResponseWriter, r *http.Request) {
	if h.semlinkReadbackIngress == nil || h.commandPlanWriter == nil {
		writeJSON(w, http.StatusServiceUnavailable, semLinkReadbackResponse{
			Error: "semlink readback ingress is not configured",
		})
		return
	}
	caller, err := h.authorizeSemLinkReadback(r)
	if err != nil {
		status := http.StatusUnauthorized
		if authErr, ok := err.(*SemLinkReadbackAuthError); ok && authErr.Status != 0 {
			status = authErr.Status
		}
		writeJSON(w, status, semLinkReadbackResponse{Error: err.Error()})
		return
	}

	var request semLinkArduPilotReadbackRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, semLinkReadbackMaxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, semLinkReadbackResponse{
			Error: "invalid semlink readback request",
		})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, semLinkReadbackResponse{
			Error: "invalid semlink readback request",
		})
		return
	}
	if err := request.validateContract(); err != nil {
		writeJSON(w, http.StatusBadRequest, semLinkReadbackResponse{Error: err.Error()})
		return
	}
	if err := validateSemLinkReadbackCallerCompanion(caller, request.companionNodeID()); err != nil {
		writeJSON(w, http.StatusForbidden, semLinkReadbackResponse{Error: err.Error()})
		return
	}

	ingressRequest, err := request.ingressRequest()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, responseForSemLinkReadbackRejection(request, err.Error()))
		return
	}
	result, plan, err := h.semlinkReadbackIngress.AdmitArduPilotReadback(r.Context(), ingressRequest)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, responseForSemLinkReadbackRejection(request, err.Error()))
		return
	}

	response := responseForSemLinkReadback(result, plan)
	response.applyCaller(caller)
	if !result.Admission.Accepted {
		writeJSON(w, http.StatusAccepted, response)
		return
	}
	if err := h.commandPlanWriter.Apply(r.Context(), plan); err != nil {
		response.Accepted = false
		response.Error = fmt.Sprintf("persist semlink readback intent: %v", err)
		writeJSON(w, http.StatusBadGateway, response)
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (h *Handler) authorizeSemLinkReadback(r *http.Request) (SemLinkReadbackCaller, error) {
	if h.semlinkReadbackAuthorizer == nil {
		return SemLinkReadbackCaller{}, nil
	}
	return h.semlinkReadbackAuthorizer(r)
}

func (r semLinkArduPilotReadbackRequest) ingressRequest() (semlinkingress.ArduPilotReadbackRequest, error) {
	if r.TTLSeconds < 0 {
		return semlinkingress.ArduPilotReadbackRequest{}, fmt.Errorf("ttl_seconds must be non-negative")
	}
	if err := r.validateV0RequiredFields(); err != nil {
		return semlinkingress.ArduPilotReadbackRequest{}, err
	}
	commandID := r.commandID()
	if commandID != semLinkReadbackCommandRequestMsg {
		return semlinkingress.ArduPilotReadbackRequest{}, fmt.Errorf(
			"unsupported command_id %d; MVP allowlist: MAV_CMD_REQUEST_MESSAGE %d",
			commandID,
			semLinkReadbackCommandRequestMsg,
		)
	}
	requestedMessageID := r.requestedMessageID()
	if requestedMessageID != semLinkReadbackMessageAutopilotVer {
		return semlinkingress.ArduPilotReadbackRequest{}, fmt.Errorf(
			"unsupported requested_message_id %d; MVP allowlist: AUTOPILOT_VERSION %d",
			requestedMessageID,
			semLinkReadbackMessageAutopilotVer,
		)
	}
	var ttl time.Duration
	if r.TTLSeconds > 0 {
		ttl = time.Duration(r.TTLSeconds) * time.Second
	}
	return semlinkingress.ArduPilotReadbackRequest{
		CompanionNodeID:    r.companionNodeID(),
		ID:                 r.ID,
		TargetAssetID:      r.TargetAssetID,
		TargetSystemID:     r.targetSystemID(),
		TargetComponentID:  r.targetComponentID(),
		CommandID:          commandID,
		RequestedMessageID: requestedMessageID,
		CorrelationID:      r.CorrelationID,
		IdempotencyKey:     r.IdempotencyKey,
		SourceRef:          r.SourceRef,
		RequestedAt:        r.requestedAt(),
		TTL:                ttl,
	}, nil
}

func (r *semLinkReadbackResponse) applyCaller(caller SemLinkReadbackCaller) {
	if r == nil || caller.ID == "" {
		return
	}
	r.AuthorizedBy = caller.ID
	r.AuthorizedCompanionNodeID = caller.CompanionNodeID
	r.AuthorizedMeshNodeID = caller.MeshNodeID
	r.AuthorityScope = caller.AuthorityScope
	r.AuthorityDomain = caller.AuthorityDomain
	r.Authenticated = caller.Authenticated
}

func validateSemLinkReadbackCallerCompanion(caller SemLinkReadbackCaller, requestCompanionNodeID string) error {
	if caller.CompanionNodeID == "" {
		return nil
	}
	if strings.TrimSpace(requestCompanionNodeID) == "" {
		return fmt.Errorf("companion_node_id is required for authenticated SemLink readback")
	}
	if strings.TrimSpace(requestCompanionNodeID) != caller.CompanionNodeID {
		return fmt.Errorf("SemLink readback companion_node_id %q does not match authenticated companion node %q", requestCompanionNodeID, caller.CompanionNodeID)
	}
	return nil
}

func responseForSemLinkReadback(
	result semlinkingress.Result,
	plan commandprojector.Plan,
) semLinkReadbackResponse {
	return semLinkReadbackResponse{
		Contract:                 semLinkReadbackContractV0,
		Accepted:                 result.Admission.Accepted,
		Status:                   semLinkReadbackStatus(result.Admission),
		RejectedReason:           semLinkReadbackRejectedReason(result.Admission.RejectedReason),
		Duplicate:                result.Admission.Duplicate,
		ExistingNativeID:         result.Admission.ExistingNativeID,
		ClaimScope:               result.ClaimScope,
		EntityID:                 entityIDForPlan(plan),
		NativeID:                 result.Intent.NativeID,
		TargetAssetID:            result.Intent.TargetAssetID,
		SourceRef:                result.Intent.SourceRef,
		CorrelationID:            result.Intent.CorrelationID,
		IdempotencyKey:           result.Intent.IdempotencyKey,
		CompanionNodeID:          companionNodeIDFromIntent(result.Intent),
		RequestedAt:              result.Intent.ObservedAt,
		ExpiresAt:                result.Intent.ExpiresAt,
		Mutations:                len(plan.Mutations),
		NativeExecutionAllowed:   result.NativeExecutionAllowed,
		CompanionTransmitAllowed: result.CompanionTransmitAllowed,
	}
}

func responseForSemLinkReadbackRejection(
	request semLinkArduPilotReadbackRequest,
	reason string,
) semLinkReadbackResponse {
	requestedAt := request.requestedAt()
	var expiresAt time.Time
	if !requestedAt.IsZero() && request.TTLSeconds > 0 {
		expiresAt = requestedAt.Add(time.Duration(request.TTLSeconds) * time.Second)
	}
	return semLinkReadbackResponse{
		Contract:                 semLinkReadbackContractV0,
		Accepted:                 false,
		Status:                   "rejected",
		RejectedReason:           semLinkReadbackRejectedReason(reason),
		Duplicate:                false,
		CorrelationID:            request.CorrelationID,
		IdempotencyKey:           request.IdempotencyKey,
		CompanionNodeID:          request.companionNodeID(),
		RequestedAt:              requestedAt,
		ExpiresAt:                expiresAt,
		Mutations:                0,
		NativeExecutionAllowed:   false,
		CompanionTransmitAllowed: false,
	}
}

func (r semLinkArduPilotReadbackRequest) validateContract() error {
	if strings.TrimSpace(r.Contract) == "" {
		return nil
	}
	if strings.TrimSpace(r.Contract) != semLinkReadbackContractV0 {
		return fmt.Errorf("unsupported SemLink readback contract %q", r.Contract)
	}
	return nil
}

func (r semLinkArduPilotReadbackRequest) validateV0RequiredFields() error {
	if r.Contract != semLinkReadbackContractV0 {
		return nil
	}
	if r.companionNodeID() == "" {
		return fmt.Errorf("companion_node_id is required for %s", semLinkReadbackContractV0)
	}
	if r.TargetSystemID == 0 {
		return fmt.Errorf("target_system_id is required for %s", semLinkReadbackContractV0)
	}
	if r.TargetComponentID == 0 {
		return fmt.Errorf("target_component_id is required for %s", semLinkReadbackContractV0)
	}
	if r.CommandID == 0 {
		return fmt.Errorf("command_id is required for %s", semLinkReadbackContractV0)
	}
	if r.RequestedMessageID == 0 {
		return fmt.Errorf("requested_message_id is required for %s", semLinkReadbackContractV0)
	}
	if r.requestedAt().IsZero() {
		return fmt.Errorf("requested_at is required for %s", semLinkReadbackContractV0)
	}
	if r.TTLSeconds <= 0 {
		return fmt.Errorf("ttl_seconds must be positive for %s", semLinkReadbackContractV0)
	}
	return nil
}

func (r semLinkArduPilotReadbackRequest) companionNodeID() string {
	if trimmed := strings.TrimSpace(r.CompanionNodeID); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(r.MeshNodeID)
}

func (r semLinkArduPilotReadbackRequest) targetSystemID() int {
	if r.TargetSystemID != 0 {
		return r.TargetSystemID
	}
	return r.VehicleSystemID
}

func (r semLinkArduPilotReadbackRequest) targetComponentID() int {
	if r.TargetComponentID != 0 {
		return r.TargetComponentID
	}
	return r.VehicleComponentID
}

func (r semLinkArduPilotReadbackRequest) commandID() int {
	if r.CommandID != 0 {
		return r.CommandID
	}
	if strings.TrimSpace(r.Action) != "" && strings.TrimSpace(r.Action) != commandprojector.SemLinkActionRequestAutopilotVersion {
		return -1
	}
	return semLinkReadbackCommandRequestMsg
}

func (r semLinkArduPilotReadbackRequest) requestedMessageID() int {
	if r.RequestedMessageID != 0 {
		return r.RequestedMessageID
	}
	return semLinkReadbackMessageAutopilotVer
}

func (r semLinkArduPilotReadbackRequest) requestedAt() time.Time {
	if !r.RequestedAt.IsZero() {
		return r.RequestedAt
	}
	return r.ObservedAt
}

func semLinkReadbackStatus(admission commandprojector.AdmissionResult) string {
	switch {
	case admission.Accepted:
		return "accepted"
	case admission.Duplicate:
		return "duplicate"
	default:
		return "rejected"
	}
}

func semLinkReadbackRejectedReason(reason string) string {
	if reason == "command intent is expired" {
		return "expired request"
	}
	return reason
}

func companionNodeIDFromIntent(intent commandprojector.Intent) string {
	return strings.TrimPrefix(strings.TrimSpace(intent.RequestedBy), "semlink:")
}

func entityIDForPlan(plan commandprojector.Plan) string {
	for _, mutation := range plan.Mutations {
		if mutation.Create.Entity != nil && mutation.Create.Entity.ID != "" {
			return mutation.Create.Entity.ID
		}
		if mutation.Update.Entity != nil && mutation.Update.Entity.ID != "" {
			return mutation.Update.Entity.ID
		}
	}
	return ""
}
