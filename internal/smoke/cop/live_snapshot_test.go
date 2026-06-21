package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
	cotcodec "github.com/c360studio/semops/pkg/adapters/cot"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
)

const (
	liveSnapshotURLEnv      = "SEMOPS_COP_SMOKE_SNAPSHOT_URL"
	liveScenarioStatusEnv   = "SEMOPS_COP_SMOKE_SCENARIO_STATUS_URL"
	liveSnapshotUDPAddrEnv  = "SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR"
	liveSnapshotCoTAddrEnv  = "SEMOPS_COP_SMOKE_COT_UDP_ADDR"
	liveSnapshotTrackIDEnv  = "SEMOPS_COP_SMOKE_EXPECTED_TRACK_ID"
	liveSnapshotCoTTrackEnv = "SEMOPS_COP_SMOKE_EXPECTED_COT_TRACK_ID"
	liveSnapshotCoTTaskEnv  = "SEMOPS_COP_SMOKE_EXPECTED_COT_TASK_ID"
	liveSnapshotCoTChatEnv  = "SEMOPS_COP_SMOKE_EXPECTED_COT_ADVISORY_ID"
	liveSnapshotHazardEnv   = "SEMOPS_COP_SMOKE_EXPECTED_HAZARD_ID"
	liveScenarioADSBEnv     = "SEMOPS_SCENARIO_ADSB_FIXTURE"
	liveSnapshotADSBHTTPEnv = "SEMOPS_COP_SMOKE_ADSB_HTTP_ENABLED"
	defaultExpectedTrackID  = "c360.edge-compose.cop.mavlink.track.system-42"
	defaultExpectedCoTTrack = "c360.edge-compose.cop.tak.track.android-alpha"
	defaultExpectedCoTTask  = "c360.edge-compose.cop.tak.task.marker-north-gate"
	defaultExpectedCoTChat  = "c360.edge-compose.cop.tak.advisory.chat-alpha-1"
	defaultExpectedHazard   = "c360.edge-compose.cop.cap.hazard_area.nws-demo-flood-warning"
	liveSnapshotPollTimeout = 30 * time.Second
)

func TestHostedCOPSnapshotReflectsMAVLinkUDP(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	udpAddr := os.Getenv(liveSnapshotUDPAddrEnv)
	if snapshotURL == "" || udpAddr == "" {
		t.Skipf("set %s and %s to run the hosted COP snapshot smoke", liveSnapshotURLEnv, liveSnapshotUDPAddrEnv)
	}
	expectedTrackID := os.Getenv(liveSnapshotTrackIDEnv)
	if expectedTrackID == "" {
		expectedTrackID = defaultExpectedTrackID
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	frames := generatedMAVLinkFrames(t)
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := sendMAVLinkFrames(ctx, udpAddr, frames); err != nil {
			lastErr = err
		}
		snapshot, err := fetchSnapshot(ctx, client, snapshotURL)
		if err != nil {
			lastErr = err
		} else if snapshotHasTrack(snapshot, expectedTrackID) {
			return
		} else {
			lastErr = fmt.Errorf("snapshot has no live graph track %s: scenario=%s tracks=%d",
				expectedTrackID, snapshot.Scenario, len(snapshot.Tracks))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect MAVLink UDP before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsCoTUDP(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	udpAddr := os.Getenv(liveSnapshotCoTAddrEnv)
	if snapshotURL == "" || udpAddr == "" {
		t.Skipf("set %s and %s to run the hosted COP CoT snapshot smoke", liveSnapshotURLEnv, liveSnapshotCoTAddrEnv)
	}
	expectedTrackID := firstNonEmpty(os.Getenv(liveSnapshotCoTTrackEnv), defaultExpectedCoTTrack)
	expectedTaskID := firstNonEmpty(os.Getenv(liveSnapshotCoTTaskEnv), defaultExpectedCoTTask)
	expectedAdvisoryID := firstNonEmpty(os.Getenv(liveSnapshotCoTChatEnv), defaultExpectedCoTChat)

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	rawEvents, err := cotcodec.MarshalEvents(cotcodec.SeedEvents(time.Now().UTC()))
	if err != nil {
		t.Fatalf("marshal cot seed events: %v", err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := sendCoTEvents(ctx, udpAddr, rawEvents); err != nil {
			lastErr = err
		}
		snapshot, err := fetchSnapshot(ctx, client, snapshotURL)
		if err != nil {
			lastErr = err
		} else if snapshotHasCoT(snapshot, expectedTrackID, expectedTaskID, expectedAdvisoryID) {
			return
		} else {
			lastErr = fmt.Errorf("snapshot missing CoT state: scenario=%s tracks=%d tasks=%d advisories=%d",
				snapshot.Scenario, len(snapshot.Tracks), len(snapshot.Tasks), len(snapshot.Advisories))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect CoT UDP before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsScenarioRunner(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	statusURL := os.Getenv(liveScenarioStatusEnv)
	if snapshotURL == "" || statusURL == "" {
		t.Skipf("set %s and %s to run the hosted COP scenario runner smoke",
			liveSnapshotURLEnv, liveScenarioStatusEnv)
	}
	expectedTrackID := firstNonEmpty(os.Getenv(liveSnapshotTrackIDEnv), defaultExpectedTrackID)
	expectedTaskID := firstNonEmpty(os.Getenv(liveSnapshotCoTTaskEnv), defaultExpectedCoTTask)
	expectedAdvisoryID := firstNonEmpty(os.Getenv(liveSnapshotCoTChatEnv), defaultExpectedCoTChat)
	expectedHazardID := firstNonEmpty(os.Getenv(liveSnapshotHazardEnv), defaultExpectedHazard)
	expectADSB, err := scenarioADSBExpectedFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		status, err := fetchScenarioStatus(ctx, client, statusURL)
		if err != nil {
			lastErr = err
		} else if status.State != "succeeded" {
			lastErr = fmt.Errorf("scenario runner state = %q, completed=%d failed=%d last_error=%q",
				status.State, status.CompletedSteps, status.FailedSteps, status.LastError)
		} else if expectADSB && status.Summary.ADSBSnapshots < 2 {
			lastErr = fmt.Errorf("scenario runner ADS-B snapshots = %d, want at least 2",
				status.Summary.ADSBSnapshots)
		} else {
			snapshot, err := fetchSnapshot(ctx, client, snapshotURL)
			if err != nil {
				lastErr = err
			} else if snapshotHasScenarioRunnerState(
				snapshot,
				expectedTrackID,
				expectedTaskID,
				expectedAdvisoryID,
				expectedHazardID,
				expectADSB,
			) {
				return
			} else {
				lastErr = fmt.Errorf("snapshot missing scenario runner state: scenario=%s tracks=%d tasks=%d advisories=%d hazards=%d expect_adsb=%v",
					snapshot.Scenario, len(snapshot.Tracks), len(snapshot.Tasks), len(snapshot.Advisories), len(snapshot.Hazards), expectADSB)
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect scenario runner before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsADSBHTTPProvider(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	if snapshotURL == "" {
		t.Skipf("set %s to run the hosted COP ADS-B snapshot smoke", liveSnapshotURLEnv)
	}
	expectADSB, err := boolFromEnv(liveSnapshotADSBHTTPEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !expectADSB {
		t.Skipf("set %s=true to run the hosted COP ADS-B snapshot smoke", liveSnapshotADSBHTTPEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		snapshot, err := fetchSnapshot(ctx, client, snapshotURL)
		if err != nil {
			lastErr = err
		} else if snapshotHasADSBTrack(snapshot) {
			return
		} else {
			lastErr = fmt.Errorf("snapshot missing ADS-B HTTP track: scenario=%s tracks=%d",
				snapshot.Scenario, len(snapshot.Tracks))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect ADS-B HTTP provider before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func generatedMAVLinkFrames(t *testing.T) [][]byte {
	t.Helper()

	generator := mavcodec.NewGenerator(42, 7)
	heartbeat, err := generator.GenerateHeartbeat(mavcodec.HeartbeatMessage{
		BaseMode:       mavcodec.ModeFlagSafetyArmed,
		SystemStatus:   mavcodec.StateActive,
		MavlinkVersion: mavcodec.Version2,
	})
	if err != nil {
		t.Fatalf("generate heartbeat: %v", err)
	}
	position, err := generator.GenerateGlobalPosition(mavcodec.PositionMessage{
		Lat: 389000001,
		Lon: -770000002,
		Vx:  321,
		Vy:  -12,
		Vz:  7,
	})
	if err != nil {
		t.Fatalf("generate position: %v", err)
	}
	return [][]byte{heartbeat, position}
}

func sendMAVLinkFrames(ctx context.Context, udpAddr string, frames [][]byte) error {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "udp", udpAddr)
	if err != nil {
		return fmt.Errorf("dial MAVLink UDP %s: %w", udpAddr, err)
	}
	defer conn.Close()

	for _, frame := range frames {
		if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return fmt.Errorf("set MAVLink UDP write deadline: %w", err)
		}
		if _, err := conn.Write(frame); err != nil {
			return fmt.Errorf("write MAVLink UDP frame: %w", err)
		}
	}
	return nil
}

func sendCoTEvents(ctx context.Context, udpAddr string, events [][]byte) error {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "udp", udpAddr)
	if err != nil {
		return fmt.Errorf("dial CoT UDP %s: %w", udpAddr, err)
	}
	defer conn.Close()

	for _, event := range events {
		if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
			return fmt.Errorf("set CoT UDP write deadline: %w", err)
		}
		if _, err := conn.Write(event); err != nil {
			return fmt.Errorf("write CoT UDP event: %w", err)
		}
	}
	return nil
}

func fetchSnapshot(ctx context.Context, client *http.Client, snapshotURL string) (copapi.Snapshot, error) {
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

func fetchScenarioStatus(ctx context.Context, client *http.Client, statusURL string) (scenarioStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return scenarioStatus{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return scenarioStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return scenarioStatus{}, fmt.Errorf("scenario status = %d", resp.StatusCode)
	}
	var status scenarioStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return scenarioStatus{}, fmt.Errorf("decode scenario status: %w", err)
	}
	return status, nil
}

type scenarioStatus struct {
	State          string `json:"state"`
	CompletedSteps int    `json:"completed_steps"`
	FailedSteps    int    `json:"failed_steps"`
	LastError      string `json:"last_error"`
	Summary        struct {
		ADSBSnapshots int `json:"adsb_snapshots"`
	} `json:"summary"`
}

func snapshotHasTrack(snapshot copapi.Snapshot, expectedTrackID string) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	for _, track := range snapshot.Tracks {
		if track.ID != expectedTrackID {
			continue
		}
		if track.Position.Lat == 0 || track.Position.Lon == 0 {
			return false
		}
		if track.Provenance.Owner == "" {
			return false
		}
		return true
	}
	return false
}

func snapshotHasScenarioRunnerState(
	snapshot copapi.Snapshot,
	expectedTrackID string,
	expectedTaskID string,
	expectedAdvisoryID string,
	expectedHazardID string,
	expectADSB bool,
) bool {
	if !snapshotHasCoT(snapshot, expectedTrackID, expectedTaskID, expectedAdvisoryID) {
		return false
	}
	if !snapshotHasHazard(snapshot, expectedHazardID) {
		return false
	}
	if expectADSB {
		return snapshotHasADSBTrack(snapshot)
	}
	return true
}

func snapshotHasCoT(snapshot copapi.Snapshot, expectedTrackID, expectedTaskID, expectedAdvisoryID string) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	if !snapshotHasTrack(snapshot, expectedTrackID) {
		return false
	}
	var hasTask bool
	for _, task := range snapshot.Tasks {
		if task.ID == expectedTaskID && task.Position != nil && task.Provenance.Owner != "" {
			hasTask = true
			break
		}
	}
	if !hasTask {
		return false
	}
	for _, advisory := range snapshot.Advisories {
		if advisory.ID == expectedAdvisoryID && advisory.Text != "" && advisory.Provenance.Owner != "" {
			return true
		}
	}
	return false
}

func snapshotHasHazard(snapshot copapi.Snapshot, expectedHazardID string) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	for _, hazard := range snapshot.Hazards {
		if hazard.ID != expectedHazardID {
			continue
		}
		if len(hazard.Geometry) == 0 || hazard.Label == "" {
			return false
		}
		if hazard.Provenance.Owner == "" {
			return false
		}
		return true
	}
	return false
}

func snapshotHasADSBTrack(snapshot copapi.Snapshot) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	for _, track := range snapshot.Tracks {
		if track.Source != "adsb" {
			continue
		}
		if track.Position.Lat == 0 || track.Position.Lon == 0 {
			return false
		}
		if track.Provenance.Owner != "semops.feed.adsb" {
			return false
		}
		return true
	}
	return false
}

func scenarioADSBExpectedFromEnv() (bool, error) {
	return boolFromEnv(liveScenarioADSBEnv)
}

func boolFromEnv(name string) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false, nil
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}
	return enabled, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
