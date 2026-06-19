package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
	mavcodec "github.com/c360studio/semops/pkg/adapters/mavlink"
)

const (
	liveSnapshotURLEnv      = "SEMOPS_COP_SMOKE_SNAPSHOT_URL"
	liveSnapshotUDPAddrEnv  = "SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR"
	liveSnapshotTrackIDEnv  = "SEMOPS_COP_SMOKE_EXPECTED_TRACK_ID"
	defaultExpectedTrackID  = "c360.edge-compose.cop.mavlink.track.system-42"
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
