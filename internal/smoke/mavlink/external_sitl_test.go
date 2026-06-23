package mavlink

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
)

const (
	sitlSnapshotURLEnv       = "SEMOPS_MAVLINK_SITL_SMOKE_SNAPSHOT_URL"
	sitlExpectedTrackIDEnv   = "SEMOPS_MAVLINK_SITL_SMOKE_EXPECTED_TRACK_ID"
	sitlRequireMotionEnv     = "SEMOPS_MAVLINK_SITL_SMOKE_REQUIRE_MOTION"
	sitlMinUpdatesEnv        = "SEMOPS_MAVLINK_SITL_SMOKE_MIN_UPDATES"
	sitlTimeoutEnv           = "SEMOPS_MAVLINK_SITL_SMOKE_TIMEOUT"
	defaultSITLTrackID       = "c360.edge-compose.cop.mavlink.track.system-1"
	defaultSITLTimeout       = 2 * time.Minute
	defaultSITLMinUpdates    = 2
	minSITLPositionDeltaDeg  = 0.0000001
	externalSITLPollInterval = 500 * time.Millisecond
)

func TestExternalSITLTelemetryCOPSnapshot(t *testing.T) {
	snapshotURL := os.Getenv(sitlSnapshotURLEnv)
	if snapshotURL == "" {
		t.Skipf("set %s to run the external MAVLink SITL COP smoke", sitlSnapshotURLEnv)
	}
	expectedTrackID := sitlFirstNonEmpty(os.Getenv(sitlExpectedTrackIDEnv), defaultSITLTrackID)
	requireMotion, err := sitlBoolFromEnv(sitlRequireMotionEnv, false)
	if err != nil {
		t.Fatal(err)
	}
	minUpdates, err := sitlIntFromEnv(sitlMinUpdatesEnv, defaultSITLMinUpdates)
	if err != nil {
		t.Fatal(err)
	}
	if minUpdates < 1 {
		t.Fatalf("%s must be at least 1", sitlMinUpdatesEnv)
	}
	timeout, err := sitlDurationFromEnv(sitlTimeoutEnv, defaultSITLTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if timeout <= 0 {
		t.Fatalf("%s must be greater than zero", sitlTimeoutEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(externalSITLPollInterval)
	defer ticker.Stop()

	var first copapi.Track
	var last copapi.Track
	var seen bool
	updates := 0
	var lastErr error
	for {
		snapshot, err := fetchSITLSnapshot(ctx, client, snapshotURL)
		if err != nil {
			lastErr = err
		} else if track, ok := findMAVLinkSITLTrack(snapshot, expectedTrackID); !ok {
			lastErr = fmt.Errorf("snapshot missing external MAVLink SITL track %s: scenario=%s tracks=%d",
				expectedTrackID, snapshot.Scenario, len(snapshot.Tracks))
		} else if err := validateMAVLinkSITLTrack(snapshot, track); err != nil {
			lastErr = err
		} else {
			if !seen {
				first = track
				last = track
				seen = true
				updates = 1
			} else if sitlTrackAdvanced(last, track) {
				last = track
				updates++
			}
			if updates >= minUpdates && (!requireMotion || sitlTrackMoved(first, last)) {
				return
			}
			lastErr = fmt.Errorf(
				"external MAVLink SITL track observed but not enough simulator churn yet: updates=%d/%d moved=%v require_motion=%v",
				updates,
				minUpdates,
				sitlTrackMoved(first, last),
				requireMotion,
			)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("external MAVLink SITL telemetry did not reach COP snapshot before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func fetchSITLSnapshot(ctx context.Context, client *http.Client, snapshotURL string) (copapi.Snapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, snapshotURL, nil)
	if err != nil {
		return copapi.Snapshot{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return copapi.Snapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return copapi.Snapshot{}, fmt.Errorf("snapshot status = %d", resp.StatusCode)
	}
	var snapshot copapi.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return copapi.Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}
	return snapshot, nil
}

func findMAVLinkSITLTrack(snapshot copapi.Snapshot, expectedTrackID string) (copapi.Track, bool) {
	if snapshot.Scenario != "phase-1-live-graph" {
		return copapi.Track{}, false
	}
	for _, track := range snapshot.Tracks {
		if track.ID == expectedTrackID && track.Source == "mavlink" {
			return track, true
		}
	}
	return copapi.Track{}, false
}

func validateMAVLinkSITLTrack(snapshot copapi.Snapshot, track copapi.Track) error {
	if !sitlSnapshotHasFeedStatus(snapshot, "feed.mavlink", "live") {
		return fmt.Errorf("feed.mavlink is not live")
	}
	if track.Position.Lat == 0 || track.Position.Lon == 0 {
		return fmt.Errorf("external MAVLink SITL track has no position: %+v", track.Position)
	}
	if track.Provenance.Owner != "semops.feed.mavlink" || track.Provenance.SourceRef == "" {
		return fmt.Errorf("external MAVLink SITL track has invalid provenance: %+v", track.Provenance)
	}
	if track.UpdatedAt.IsZero() || track.Provenance.Observed.IsZero() {
		return fmt.Errorf("external MAVLink SITL track has no observation time: updated=%s observed=%s",
			track.UpdatedAt, track.Provenance.Observed)
	}
	if track.Velocity == "" {
		return fmt.Errorf("external MAVLink SITL track has no velocity evidence")
	}
	return nil
}

func sitlSnapshotHasFeedStatus(snapshot copapi.Snapshot, feedID string, statuses ...string) bool {
	for _, feed := range snapshot.Feeds {
		if feed.ID != feedID {
			continue
		}
		for _, status := range statuses {
			if feed.Status == status {
				return true
			}
		}
	}
	return false
}

func sitlTrackAdvanced(previous, next copapi.Track) bool {
	if next.UpdatedAt.After(previous.UpdatedAt) || next.Provenance.Observed.After(previous.Provenance.Observed) {
		return true
	}
	return sitlTrackMoved(previous, next)
}

func sitlTrackMoved(first, last copapi.Track) bool {
	return math.Abs(first.Position.Lat-last.Position.Lat) >= minSITLPositionDeltaDeg ||
		math.Abs(first.Position.Lon-last.Position.Lon) >= minSITLPositionDeltaDeg
}

func sitlFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sitlBoolFromEnv(name string, defaultValue bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes", "on", "ON", "On":
		return true, nil
	case "0", "false", "FALSE", "False", "no", "NO", "No", "off", "OFF", "Off":
		return false, nil
	default:
		return false, fmt.Errorf("%s=%q is not a boolean", name, value)
	}
}

func sitlIntFromEnv(name string, defaultValue int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not an integer: %w", name, value, err)
	}
	return parsed, nil
}

func sitlDurationFromEnv(name string, defaultValue time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s=%q is not a duration: %w", name, value, err)
	}
	return parsed, nil
}
