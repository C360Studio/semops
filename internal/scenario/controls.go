package scenario

import "slices"

type ControlAction string

const (
	ControlActionStart  ControlAction = "start"
	ControlActionReset  ControlAction = "reset"
	ControlActionPause  ControlAction = "pause"
	ControlActionResume ControlAction = "resume"
)

const scenarioControlBlockedReason = "scenario controls require a reviewed operator_scenario_control checkpoint and an implemented control executor"

type ControlCapabilities struct {
	Enabled            bool            `json:"enabled"`
	State              string          `json:"state"`
	Reason             string          `json:"reason"`
	SupportedActions   []ControlAction `json:"supported_actions"`
	RequiredClaimScope ClaimScope      `json:"required_claim_scope"`
	ScenarioID         string          `json:"scenario_id,omitempty"`
	CheckpointID       string          `json:"checkpoint_id,omitempty"`
	CheckpointState    string          `json:"checkpoint_state,omitempty"`
}

type ControlRequest struct {
	Action ControlAction `json:"action"`
}

type ControlResult struct {
	Accepted bool                `json:"accepted"`
	Action   ControlAction       `json:"action,omitempty"`
	State    string              `json:"state"`
	Reason   string              `json:"reason"`
	Controls ControlCapabilities `json:"controls"`
}

func EvaluateControlCapabilities(status Status) ControlCapabilities {
	capabilities := ControlCapabilities{
		Enabled:            false,
		State:              "blocked",
		Reason:             scenarioControlBlockedReason,
		SupportedActions:   supportedControlActions(),
		RequiredClaimScope: ClaimOperatorControl,
		ScenarioID:         status.ScenarioID,
	}
	for _, checkpoint := range status.Checkpoints {
		if checkpoint.ClaimScope != ClaimOperatorControl {
			continue
		}
		capabilities.CheckpointID = checkpoint.ID
		capabilities.CheckpointState = string(checkpoint.State)
		return capabilities
	}
	return capabilities
}

func RejectControlRequest(status Status, action ControlAction) ControlResult {
	return ControlResult{
		Accepted: false,
		Action:   action,
		State:    "blocked",
		Reason:   scenarioControlBlockedReason,
		Controls: EvaluateControlCapabilities(status),
	}
}

func ValidControlAction(action ControlAction) bool {
	return slices.Contains(supportedControlActions(), action)
}

func supportedControlActions() []ControlAction {
	return []ControlAction{
		ControlActionStart,
		ControlActionReset,
		ControlActionPause,
		ControlActionResume,
	}
}
