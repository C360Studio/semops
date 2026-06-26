package scenario

import "testing"

func TestEvaluateProductCheckpointReadyForReadback(t *testing.T) {
	manifest := checkpointEvaluationManifest(t)
	status := Status{
		ScenarioID:     Phase1HADRScenarioID,
		State:          StateSucceeded,
		IngressMode:    IngressModeFeedBoundary,
		CompletedSteps: 6,
		Summary: Summary{
			MAVLinkFrames:          2,
			CoTEvents:              4,
			FeedBoundaryDeliveries: 6,
		},
	}

	evaluations := EvaluateCheckpoints(manifest, status)
	evaluation, ok := CheckpointEvaluationForID(evaluations, "hadr-shared-airspace-product-e2e")
	if !ok {
		t.Fatalf("product checkpoint missing: %+v", evaluations)
	}
	if evaluation.State != CheckpointReadyForReadback {
		t.Fatalf("product checkpoint state = %q messages=%v", evaluation.State, evaluation.Messages)
	}
}

func TestEvaluateProductCheckpointRejectsGraphMutationEvidence(t *testing.T) {
	manifest := checkpointEvaluationManifest(t)
	status := Status{
		ScenarioID:     Phase1HADRScenarioID,
		State:          StateSucceeded,
		IngressMode:    IngressModeFeedBoundary,
		CompletedSteps: 1,
		Summary: Summary{
			MAVLinkFrames:                 1,
			FeedBoundaryDeliveries:        1,
			Mutations:                     1,
			ContractGraphMutationAttempts: 1,
		},
	}

	evaluations := EvaluateCheckpoints(manifest, status)
	evaluation, ok := CheckpointEvaluationForID(evaluations, "hadr-shared-airspace-product-e2e")
	if !ok {
		t.Fatalf("product checkpoint missing: %+v", evaluations)
	}
	if evaluation.State != CheckpointFailed {
		t.Fatalf("product checkpoint state = %q messages=%v", evaluation.State, evaluation.Messages)
	}
}

func TestEvaluateUnownedComponentCheckpointRemainsDeclared(t *testing.T) {
	manifest := checkpointEvaluationManifest(t)
	status := Status{
		ScenarioID:     Phase1HADRScenarioID,
		State:          StateSucceeded,
		IngressMode:    IngressModeFeedBoundary,
		CompletedSteps: 6,
		Summary: Summary{
			MAVLinkFrames:          2,
			CoTEvents:              4,
			FeedBoundaryDeliveries: 6,
		},
	}

	evaluations := EvaluateCheckpoints(manifest, status)
	evaluation, ok := CheckpointEvaluationForID(evaluations, "sapient-preflight-decoded-stream")
	if !ok {
		t.Fatalf("SAPIENT checkpoint missing: %+v", evaluations)
	}
	if evaluation.State != CheckpointDeclared {
		t.Fatalf("SAPIENT checkpoint state = %q messages=%v", evaluation.State, evaluation.Messages)
	}
}

func TestNewRunnerRejectsCheckpointManifestForWrongScenario(t *testing.T) {
	fixture := Fixture{ID: "other-scenario"}
	_, err := NewRunner(Config{
		Fixture: fixture,
		Checkpoints: CheckpointManifest{
			Version:    CheckpointManifestVersion,
			ScenarioID: Phase1HADRScenarioID,
			Checkpoints: []ScenarioCheckpoint{{
				ID:          "contract",
				ClaimScope:  ClaimContractReplay,
				IngressMode: IngressModeDirectGraphContract,
				Boundary:    CheckpointBoundary{Kind: BoundaryDirectGraph},
				AllowedOwners: []string{
					"semops.feed.mavlink",
				},
			}},
		},
	})
	if err == nil {
		t.Fatal("expected runner to reject checkpoint manifest for a different scenario")
	}
}

func checkpointEvaluationManifest(t *testing.T) CheckpointManifest {
	t.Helper()
	manifest, err := LoadCheckpointManifest("../../scenarios/phase1-hadr.checkpoints.json")
	if err != nil {
		t.Fatalf("load checkpoint manifest: %v", err)
	}
	return manifest
}
