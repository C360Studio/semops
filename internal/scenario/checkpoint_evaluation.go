package scenario

import (
	"fmt"
	"slices"
)

type CheckpointEvaluationState string

const (
	CheckpointDeclared         CheckpointEvaluationState = "declared"
	CheckpointPending          CheckpointEvaluationState = "pending"
	CheckpointReadyForReadback CheckpointEvaluationState = "ready_for_readback"
	CheckpointFailed           CheckpointEvaluationState = "failed"
)

type CheckpointEvaluation struct {
	ID          string                    `json:"id"`
	Label       string                    `json:"label,omitempty"`
	ClaimScope  ClaimScope                `json:"claim_scope"`
	IngressMode IngressMode               `json:"ingress_mode"`
	Boundary    BoundaryKind              `json:"boundary"`
	State       CheckpointEvaluationState `json:"state"`
	Messages    []string                  `json:"messages,omitempty"`
}

func EvaluateCheckpoints(manifest CheckpointManifest, status Status) []CheckpointEvaluation {
	if len(manifest.Checkpoints) == 0 {
		return nil
	}
	evaluations := make([]CheckpointEvaluation, 0, len(manifest.Checkpoints))
	for _, checkpoint := range manifest.Checkpoints {
		evaluations = append(evaluations, evaluateCheckpoint(checkpoint, status))
	}
	return evaluations
}

func validateCheckpointManifestForFixture(manifest CheckpointManifest, fixture Fixture) error {
	if manifest.Version == "" && manifest.ScenarioID == "" && len(manifest.Checkpoints) == 0 {
		return nil
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	if manifest.ScenarioID != fixture.ID {
		return fmt.Errorf("scenario checkpoint manifest scenario_id = %q, want fixture id %q",
			manifest.ScenarioID,
			fixture.ID)
	}
	return nil
}

func evaluateCheckpoint(checkpoint ScenarioCheckpoint, status Status) CheckpointEvaluation {
	evaluation := CheckpointEvaluation{
		ID:          checkpoint.ID,
		Label:       checkpoint.Label,
		ClaimScope:  checkpoint.ClaimScope,
		IngressMode: checkpoint.IngressMode,
		Boundary:    checkpoint.Boundary.Kind,
		State:       CheckpointDeclared,
	}
	if !checkpointAppliesToStatus(checkpoint, status) {
		evaluation.Messages = append(evaluation.Messages,
			fmt.Sprintf("checkpoint is not active for runner ingress %q", status.IngressMode))
		return evaluation
	}

	switch checkpoint.ClaimScope {
	case ClaimProductE2E:
		return evaluateProductCheckpoint(checkpoint, status, evaluation)
	case ClaimContractReplay:
		return evaluateContractCheckpoint(status, evaluation)
	case ClaimComponentPreflight:
		evaluation.Messages = append(evaluation.Messages,
			"component preflight evidence is evaluated by runtime and metrics readback")
		return evaluation
	default:
		evaluation.Messages = append(evaluation.Messages,
			"checkpoint claim requires external readback or review evidence outside the scenario runner")
		return evaluation
	}
}

func checkpointAppliesToStatus(checkpoint ScenarioCheckpoint, status Status) bool {
	if status.IngressMode != checkpoint.IngressMode {
		return false
	}
	switch checkpoint.ClaimScope {
	case ClaimProductE2E, ClaimContractReplay:
		return true
	case ClaimComponentPreflight:
		return checkpointTouchesRunnerFeed(checkpoint, status)
	default:
		return false
	}
}

func evaluateProductCheckpoint(
	checkpoint ScenarioCheckpoint,
	status Status,
	evaluation CheckpointEvaluation,
) CheckpointEvaluation {
	if status.State == StateFailed {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages, firstNonEmptyString(status.LastError, "scenario runner failed"))
		return evaluation
	}
	if status.State != StateSucceeded {
		evaluation.State = CheckpointPending
		evaluation.Messages = append(evaluation.Messages,
			fmt.Sprintf("waiting for scenario runner to succeed; state=%q", status.State))
		return evaluation
	}
	if checkpoint.IngressMode != IngressModeFeedBoundary {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages, "product checkpoint must use feed-boundary ingress")
		return evaluation
	}
	if status.Summary.Mutations != 0 || status.Summary.ContractGraphMutationAttempts != 0 {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages,
			fmt.Sprintf("product checkpoint reported graph mutation evidence: mutations=%d contract_graph_mutation_attempts=%d",
				status.Summary.Mutations,
				status.Summary.ContractGraphMutationAttempts))
		return evaluation
	}
	if status.CompletedSteps == 0 {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages, "product checkpoint has no completed feed-boundary steps")
		return evaluation
	}
	if status.Summary.FeedBoundaryDeliveries != status.CompletedSteps {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages,
			fmt.Sprintf("feed-boundary deliveries=%d completed_steps=%d",
				status.Summary.FeedBoundaryDeliveries,
				status.CompletedSteps))
		return evaluation
	}

	evaluation.State = CheckpointReadyForReadback
	evaluation.Messages = append(evaluation.Messages,
		"runner evidence satisfied; stack smoke must verify COP, runtime, and Prometheus readback")
	return evaluation
}

func evaluateContractCheckpoint(status Status, evaluation CheckpointEvaluation) CheckpointEvaluation {
	if status.State == StateFailed {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages, firstNonEmptyString(status.LastError, "scenario runner failed"))
		return evaluation
	}
	if status.State != StateSucceeded {
		evaluation.State = CheckpointPending
		evaluation.Messages = append(evaluation.Messages,
			fmt.Sprintf("waiting for scenario runner to succeed; state=%q", status.State))
		return evaluation
	}
	if status.Summary.Mutations == 0 || status.Summary.ContractGraphMutationAttempts == 0 {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages, "contract replay produced no graph mutation evidence")
		return evaluation
	}
	if status.Summary.ContractGraphMutationAttempts != status.Summary.Mutations {
		evaluation.State = CheckpointFailed
		evaluation.Messages = append(evaluation.Messages,
			fmt.Sprintf("contract_graph_mutation_attempts=%d mutations=%d",
				status.Summary.ContractGraphMutationAttempts,
				status.Summary.Mutations))
		return evaluation
	}
	evaluation.State = CheckpointReadyForReadback
	evaluation.Messages = append(evaluation.Messages, "contract replay evidence satisfied")
	return evaluation
}

func checkpointTouchesRunnerFeed(checkpoint ScenarioCheckpoint, status Status) bool {
	active := runnerFeedsFromStatus(status)
	for _, feed := range checkpoint.Boundary.Feeds {
		if _, ok := active[feed]; ok {
			return true
		}
	}
	return false
}

func runnerFeedsFromStatus(status Status) map[string]struct{} {
	feeds := map[string]struct{}{}
	if status.Summary.MAVLinkFrames > 0 {
		feeds["mavlink"] = struct{}{}
	}
	if status.Summary.CoTEvents > 0 {
		feeds["tak-cot"] = struct{}{}
		feeds["tak"] = struct{}{}
	}
	if status.Summary.CAPAlerts > 0 {
		feeds["cap"] = struct{}{}
		feeds["cap-edxl"] = struct{}{}
	}
	if status.Summary.ADSBSnapshots > 0 {
		feeds["adsb"] = struct{}{}
	}
	return feeds
}

func CheckpointEvaluationForID(evaluations []CheckpointEvaluation, id string) (CheckpointEvaluation, bool) {
	for _, evaluation := range evaluations {
		if evaluation.ID == id {
			return evaluation, true
		}
	}
	return CheckpointEvaluation{}, false
}

func CheckpointByClaim(manifest CheckpointManifest, scope ClaimScope) (ScenarioCheckpoint, bool) {
	for _, checkpoint := range manifest.Checkpoints {
		if checkpoint.ClaimScope == scope {
			return checkpoint, true
		}
	}
	return ScenarioCheckpoint{}, false
}

func CheckpointExpectsFeed(checkpoint ScenarioCheckpoint, feed string) bool {
	return slices.Contains(checkpoint.ExpectedCOP.Feeds, feed)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
