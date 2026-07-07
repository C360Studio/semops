package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Error                    string `json:"error,omitempty"`
}

func (h *Handler) admitSemLinkArduPilotReadback(w http.ResponseWriter, r *http.Request) {
	if h.semlinkReadbackIngress == nil || h.commandPlanWriter == nil {
		writeJSON(w, http.StatusServiceUnavailable, semLinkReadbackResponse{
			Error: "semlink readback ingress is not configured",
		})
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
