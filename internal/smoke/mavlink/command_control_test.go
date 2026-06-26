package mavlink

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
)

const (
	commandSnapshotURLEnv            = "SEMOPS_MAVLINK_COMMAND_SMOKE_SNAPSHOT_URL"
	commandExpectedAckTaskIDEnv      = "SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_TASK_ID"
	commandExpectedAckStatusEnv      = "SEMOPS_MAVLINK_COMMAND_EXPECTED_ACK_STATUS"
	commandExpectedTaskTargetIDEnv   = "SEMOPS_MAVLINK_COMMAND_EXPECTED_TASK_TARGET_ID"
	commandPostStateTrackIDEnv       = "SEMOPS_MAVLINK_COMMAND_POST_STATE_TRACK_ID"
	commandStartedAtEnv              = "SEMOPS_MAVLINK_COMMAND_STARTED_AT"
	commandTimeoutEnv                = "SEMOPS_MAVLINK_COMMAND_SMOKE_TIMEOUT"
	commandPostStateMinUpdatesEnv    = "SEMOPS_MAVLINK_COMMAND_POST_STATE_MIN_UPDATES"
	commandPostStateRequireMotionEnv = "SEMOPS_MAVLINK_COMMAND_POST_STATE_REQUIRE_MOTION"
	defaultCommandAckStatus          = "accepted"
)

func TestCommandControlSimulatorGateCOPSnapshot(t *testing.T) {
	snapshotURL := os.Getenv(commandSnapshotURLEnv)
	if snapshotURL == "" {
		t.Skipf("set %s to run the MAVLink command-control simulator COP smoke", commandSnapshotURLEnv)
	}
	expectedAckTaskID := os.Getenv(commandExpectedAckTaskIDEnv)
	if expectedAckTaskID == "" {
		t.Fatalf("%s is required", commandExpectedAckTaskIDEnv)
	}
	postStateTrackID := os.Getenv(commandPostStateTrackIDEnv)
	if postStateTrackID == "" {
		t.Fatalf("%s is required", commandPostStateTrackIDEnv)
	}
	startedAt, err := commandStartedAt()
	if err != nil {
		t.Fatal(err)
	}
	timeout, err := sitlDurationFromEnv(commandTimeoutEnv, defaultSITLTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if timeout <= 0 {
		t.Fatalf("%s must be greater than zero", commandTimeoutEnv)
	}
	minUpdates, err := sitlIntFromEnv(commandPostStateMinUpdatesEnv, defaultSITLMinUpdates)
	if err != nil {
		t.Fatal(err)
	}
	if minUpdates < 1 {
		t.Fatalf("%s must be at least 1", commandPostStateMinUpdatesEnv)
	}
	requireMotion, err := sitlBoolFromEnv(commandPostStateRequireMotionEnv, false)
	if err != nil {
		t.Fatal(err)
	}
	expectedStatuses := commandExpectedStatuses(os.Getenv(commandExpectedAckStatusEnv))
	if len(expectedStatuses) == 0 {
		t.Fatalf("%s must include at least one status", commandExpectedAckStatusEnv)
	}
	expectedTargetID := os.Getenv(commandExpectedTaskTargetIDEnv)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(externalSITLPollInterval)
	defer ticker.Stop()

	var first copapi.Track
	var last copapi.Track
	var seenTrack bool
	updates := 0
	var lastErr error
	for {
		snapshot, err := fetchSITLSnapshot(ctx, client, snapshotURL)
		if err != nil {
			lastErr = err
		} else if task, ok := findMAVLinkCommandTask(snapshot, expectedAckTaskID); !ok {
			lastErr = fmt.Errorf("snapshot missing MAVLink command ACK task %s: scenario=%s tasks=%d",
				expectedAckTaskID, snapshot.Scenario, len(snapshot.Tasks))
		} else if err := validateMAVLinkCommandACKTask(task, expectedStatuses, expectedTargetID, startedAt); err != nil {
			lastErr = err
		} else if track, ok := findMAVLinkSITLTrack(snapshot, postStateTrackID); !ok {
			lastErr = fmt.Errorf("snapshot missing post-command MAVLink track %s: scenario=%s tracks=%d",
				postStateTrackID, snapshot.Scenario, len(snapshot.Tracks))
		} else if err := validateMAVLinkPostCommandTrack(snapshot, track, startedAt); err != nil {
			lastErr = err
		} else {
			if !seenTrack {
				first = track
				last = track
				seenTrack = true
				updates = 1
			} else if sitlTrackAdvanced(last, track) {
				last = track
				updates++
			}
			if updates >= minUpdates && (!requireMotion || sitlTrackMoved(first, last)) {
				return
			}
			lastErr = fmt.Errorf(
				"MAVLink command ACK observed but post-command state has not advanced enough: updates=%d/%d moved=%v require_motion=%v",
				updates,
				minUpdates,
				sitlTrackMoved(first, last),
				requireMotion,
			)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("MAVLink command-control simulator gate did not pass before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func commandStartedAt() (time.Time, error) {
	value := os.Getenv(commandStartedAtEnv)
	if value == "" {
		return time.Now().UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s=%q is not RFC3339: %w", commandStartedAtEnv, value, err)
	}
	return parsed.UTC(), nil
}

func commandExpectedStatuses(value string) map[string]struct{} {
	if value == "" {
		value = defaultCommandAckStatus
	}
	statuses := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		status := strings.TrimSpace(part)
		if status == "" {
			continue
		}
		statuses[status] = struct{}{}
	}
	return statuses
}

func findMAVLinkCommandTask(snapshot copapi.Snapshot, expectedTaskID string) (copapi.Task, bool) {
	if snapshot.Scenario != "phase-1-live-graph" {
		return copapi.Task{}, false
	}
	for _, task := range snapshot.Tasks {
		if task.ID == expectedTaskID && task.Source == "mavlink" {
			return task, true
		}
	}
	return copapi.Task{}, false
}

func validateMAVLinkCommandACKTask(
	task copapi.Task,
	expectedStatuses map[string]struct{},
	expectedTargetID string,
	startedAt time.Time,
) error {
	if task.Kind != "mavlink.command_ack" {
		return fmt.Errorf("command task kind = %q, want mavlink.command_ack", task.Kind)
	}
	if _, ok := expectedStatuses[task.Status]; !ok {
		return fmt.Errorf("command ACK status = %q, want one of %s", task.Status, commandStatusList(expectedStatuses))
	}
	if expectedTargetID != "" && task.TargetID != expectedTargetID {
		return fmt.Errorf("command ACK target = %q, want %q", task.TargetID, expectedTargetID)
	}
	if task.Provenance.Owner != "semops.feed.mavlink" || task.Provenance.SourceRef == "" {
		return fmt.Errorf("command ACK task has invalid provenance: %+v", task.Provenance)
	}
	if task.UpdatedAt.IsZero() || task.Provenance.Observed.IsZero() {
		return fmt.Errorf("command ACK task has no observation time: updated=%s observed=%s",
			task.UpdatedAt, task.Provenance.Observed)
	}
	if task.UpdatedAt.Before(startedAt) && task.Provenance.Observed.Before(startedAt) {
		return fmt.Errorf("command ACK task is stale: started_at=%s updated=%s observed=%s",
			startedAt, task.UpdatedAt, task.Provenance.Observed)
	}
	if task.Description == "" {
		return fmt.Errorf("command ACK task is missing native command description")
	}
	return nil
}

func validateMAVLinkPostCommandTrack(snapshot copapi.Snapshot, track copapi.Track, startedAt time.Time) error {
	if err := validateMAVLinkSITLTrack(snapshot, track); err != nil {
		return err
	}
	if track.UpdatedAt.Before(startedAt) && track.Provenance.Observed.Before(startedAt) {
		return fmt.Errorf("post-command track is stale: started_at=%s updated=%s observed=%s",
			startedAt, track.UpdatedAt, track.Provenance.Observed)
	}
	return nil
}

func commandStatusList(statuses map[string]struct{}) string {
	ordered := make([]string, 0, len(statuses))
	for status := range statuses {
		ordered = append(ordered, status)
	}
	sort.Strings(ordered)
	return strings.Join(ordered, ",")
}
