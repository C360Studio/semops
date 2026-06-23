package command

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestLifecycleFixtureRecordsMatchCommittedFixture(t *testing.T) {
	generated, err := MarshalLifecycleReplay(LifecycleFixtureRecords())
	if err != nil {
		t.Fatalf("marshal lifecycle fixture: %v", err)
	}
	committed, err := os.ReadFile(commandLifecycleFixturePath())
	if err != nil {
		t.Fatalf("read committed fixture: %v", err)
	}
	if !bytes.Equal(generated, committed) {
		t.Fatalf("generated command lifecycle fixture drifted from %s", commandLifecycleFixturePath())
	}
}

func TestLoadLifecycleReplayAppliesCommandLifecycleStory(t *testing.T) {
	records, err := LoadLifecycleReplay(commandLifecycleFixturePath())
	if err != nil {
		t.Fatalf("load lifecycle replay: %v", err)
	}
	if len(records) != 10 {
		t.Fatalf("records = %d, want 10", len(records))
	}

	projector := NewProjector(Config{OwnerTokens: testOwnerTokens("test")})
	current := map[string]Intent{}

	for _, record := range records {
		switch record.Kind {
		case ReplayIntent:
			intent := *record.Intent
			plan, err := projector.ProjectIntent(intent)
			if err != nil {
				t.Fatalf("%s project intent: %v", record.Ref, err)
			}
			if marked := projector.MarkBornForPlan(plan); marked != 1 {
				t.Fatalf("%s marked births = %d, want 1", record.Ref, marked)
			}
			requirePlanStatus(t, plan, intent.NativeID, intent.Status)
			current[intent.NativeID] = intent
		case ReplayNativeStatus:
			evidence := *record.NativeStatus
			intent := requireCurrentIntent(t, current, evidence.NativeID)
			update, err := ReconcileNativeStatus(intent, evidence)
			if err != nil {
				t.Fatalf("%s reconcile native status: %v", record.Ref, err)
			}
			plan, err := projector.ProjectStatusUpdate(update)
			if err != nil {
				t.Fatalf("%s project native status: %v", record.Ref, err)
			}
			requirePlanStatus(t, plan, evidence.NativeID, update.Status)
			current[evidence.NativeID] = applyStatusUpdate(intent, update)
		case ReplayCancellation:
			request := *record.Cancellation
			intent := requireCurrentIntent(t, current, request.NativeID)
			cancelIntent, err := BuildCancellationIntent(intent, request)
			if err != nil {
				t.Fatalf("%s build cancellation intent: %v", record.Ref, err)
			}
			plan, err := projector.ProjectIntent(cancelIntent)
			if err != nil {
				t.Fatalf("%s project cancellation intent: %v", record.Ref, err)
			}
			requirePlanStatus(t, plan, request.NativeID, StatusCancelRequested)
			current[request.NativeID] = cancelIntent
		case ReplayDeadline:
			evidence := *record.Deadline
			intent := requireCurrentIntent(t, current, evidence.NativeID)
			update, err := ReconcileDeadline(intent, evidence)
			if err != nil {
				t.Fatalf("%s reconcile deadline: %v", record.Ref, err)
			}
			plan, err := projector.ProjectStatusUpdate(update)
			if err != nil {
				t.Fatalf("%s project deadline: %v", record.Ref, err)
			}
			requirePlanStatus(t, plan, evidence.NativeID, update.Status)
			current[evidence.NativeID] = applyStatusUpdate(intent, update)
		default:
			t.Fatalf("%s unsupported replay kind %q", record.Ref, record.Kind)
		}
	}

	requireIntentStatus(t, current, "csapi-command-route-42", StatusCancelled)
	requireIntentStatus(t, current, "csapi-command-survey-7", StatusTimeout)
	requireIntentStatus(t, current, "csapi-command-stale-1", StatusExpired)
}

func commandLifecycleFixturePath() string {
	return filepath.Join("..", "..", "..", "fixtures", "command", "lifecycle", "hadr-command.jsonl")
}

func requireCurrentIntent(t *testing.T, current map[string]Intent, nativeID string) Intent {
	t.Helper()
	intent, ok := current[nativeID]
	if !ok {
		t.Fatalf("missing current intent %q", nativeID)
	}
	return intent
}

func applyStatusUpdate(intent Intent, update StatusUpdate) Intent {
	intent.Status = update.Status
	intent.Description = update.Description
	intent.ObservedAt = update.ObservedAt
	intent.Source = update.Source
	intent.SourceRef = update.SourceRef
	return intent
}

func requireIntentStatus(t *testing.T, current map[string]Intent, nativeID string, status string) {
	t.Helper()
	intent := requireCurrentIntent(t, current, nativeID)
	if intent.Status != status {
		t.Fatalf("%s status = %q, want %q", nativeID, intent.Status, status)
	}
}
