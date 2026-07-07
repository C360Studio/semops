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

const semLinkReadbackMaxBodyBytes = 8192

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
	MeshNodeID         string    `json:"mesh_node_id"`
	ID                 string    `json:"id,omitempty"`
	TargetAssetID      string    `json:"target_asset_id"`
	VehicleSystemID    int       `json:"vehicle_system_id"`
	VehicleComponentID int       `json:"vehicle_component_id,omitempty"`
	Action             string    `json:"action,omitempty"`
	CorrelationID      string    `json:"correlation_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	SourceRef          string    `json:"source_ref,omitempty"`
	ObservedAt         time.Time `json:"observed_at,omitempty"`
	TTLSeconds         int       `json:"ttl_seconds,omitempty"`
}

type semLinkReadbackResponse struct {
	Accepted                 bool   `json:"accepted"`
	RejectedReason           string `json:"rejected_reason,omitempty"`
	Duplicate                bool   `json:"duplicate,omitempty"`
	ExistingNativeID         string `json:"existing_native_id,omitempty"`
	ClaimScope               string `json:"claim_scope,omitempty"`
	EntityID                 string `json:"entity_id,omitempty"`
	NativeID                 string `json:"native_id,omitempty"`
	TargetAssetID            string `json:"target_asset_id,omitempty"`
	SourceRef                string `json:"source_ref,omitempty"`
	Mutations                int    `json:"mutations"`
	NativeExecutionAllowed   bool   `json:"native_execution_allowed"`
	CompanionTransmitAllowed bool   `json:"companion_transmit_allowed"`
	AuthorizedBy             string `json:"authorized_by,omitempty"`
	AuthorizedMeshNodeID     string `json:"authorized_mesh_node_id,omitempty"`
	AuthorityScope           string `json:"authority_scope,omitempty"`
	AuthorityDomain          string `json:"authority_domain,omitempty"`
	Authenticated            bool   `json:"authenticated,omitempty"`
	Error                    string `json:"error,omitempty"`
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
	if err := validateSemLinkReadbackCallerMesh(caller, request.MeshNodeID); err != nil {
		writeJSON(w, http.StatusForbidden, semLinkReadbackResponse{Error: err.Error()})
		return
	}

	ingressRequest, err := request.ingressRequest()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, semLinkReadbackResponse{Error: err.Error()})
		return
	}
	result, plan, err := h.semlinkReadbackIngress.AdmitArduPilotReadback(r.Context(), ingressRequest)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, semLinkReadbackResponse{Error: err.Error()})
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
	var ttl time.Duration
	if r.TTLSeconds > 0 {
		ttl = time.Duration(r.TTLSeconds) * time.Second
	}
	return semlinkingress.ArduPilotReadbackRequest{
		MeshNodeID:       r.MeshNodeID,
		ID:               r.ID,
		TargetAssetID:    r.TargetAssetID,
		VehicleSystemID:  r.VehicleSystemID,
		VehicleComponent: r.VehicleComponentID,
		Action:           r.Action,
		CorrelationID:    r.CorrelationID,
		IdempotencyKey:   r.IdempotencyKey,
		SourceRef:        r.SourceRef,
		ObservedAt:       r.ObservedAt,
		TTL:              ttl,
	}, nil
}

func (r *semLinkReadbackResponse) applyCaller(caller SemLinkReadbackCaller) {
	if r == nil || caller.ID == "" {
		return
	}
	r.AuthorizedBy = caller.ID
	r.AuthorizedMeshNodeID = caller.MeshNodeID
	r.AuthorityScope = caller.AuthorityScope
	r.AuthorityDomain = caller.AuthorityDomain
	r.Authenticated = caller.Authenticated
}

func validateSemLinkReadbackCallerMesh(caller SemLinkReadbackCaller, requestMeshNodeID string) error {
	if caller.MeshNodeID == "" {
		return nil
	}
	if strings.TrimSpace(requestMeshNodeID) == "" {
		return fmt.Errorf("mesh_node_id is required for authenticated SemLink readback")
	}
	if strings.TrimSpace(requestMeshNodeID) != caller.MeshNodeID {
		return fmt.Errorf("SemLink readback mesh_node_id %q does not match authenticated mesh node %q", requestMeshNodeID, caller.MeshNodeID)
	}
	return nil
}

func responseForSemLinkReadback(
	result semlinkingress.Result,
	plan commandprojector.Plan,
) semLinkReadbackResponse {
	return semLinkReadbackResponse{
		Accepted:                 result.Admission.Accepted,
		RejectedReason:           result.Admission.RejectedReason,
		Duplicate:                result.Admission.Duplicate,
		ExistingNativeID:         result.Admission.ExistingNativeID,
		ClaimScope:               result.ClaimScope,
		EntityID:                 entityIDForPlan(plan),
		NativeID:                 result.Intent.NativeID,
		TargetAssetID:            result.Intent.TargetAssetID,
		SourceRef:                result.Intent.SourceRef,
		Mutations:                len(plan.Mutations),
		NativeExecutionAllowed:   result.NativeExecutionAllowed,
		CompanionTransmitAllowed: result.CompanionTransmitAllowed,
	}
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
