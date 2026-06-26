package cop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
	liveSnapshotURLEnv            = "SEMOPS_COP_SMOKE_SNAPSHOT_URL"
	liveRuntimeURLEnv             = "SEMOPS_COP_SMOKE_RUNTIME_URL"
	liveScenarioStatusEnv         = "SEMOPS_COP_SMOKE_SCENARIO_STATUS_URL"
	liveComponentMetricsEnv       = "SEMOPS_COP_SMOKE_COMPONENT_METRICS_URL"
	liveSnapshotUDPAddrEnv        = "SEMOPS_COP_SMOKE_MAVLINK_UDP_ADDR"
	liveSnapshotCoTAddrEnv        = "SEMOPS_COP_SMOKE_COT_UDP_ADDR"
	liveSnapshotTrackIDEnv        = "SEMOPS_COP_SMOKE_EXPECTED_TRACK_ID"
	liveSnapshotCoTTrackEnv       = "SEMOPS_COP_SMOKE_EXPECTED_COT_TRACK_ID"
	liveSnapshotCoTTaskEnv        = "SEMOPS_COP_SMOKE_EXPECTED_COT_TASK_ID"
	liveSnapshotCoTChatEnv        = "SEMOPS_COP_SMOKE_EXPECTED_COT_ADVISORY_ID"
	liveSnapshotHazardEnv         = "SEMOPS_COP_SMOKE_EXPECTED_HAZARD_ID"
	liveSnapshotCAPHTTPEnv        = "SEMOPS_COP_SMOKE_CAP_HTTP_ENABLED"
	liveScenarioADSBEnv           = "SEMOPS_SCENARIO_ADSB_FIXTURE"
	liveSnapshotADSBHTTPEnv       = "SEMOPS_COP_SMOKE_ADSB_HTTP_ENABLED"
	liveSnapshotSAPIENTEnv        = "SEMOPS_COP_SMOKE_SAPIENT_HTTP_ENABLED"
	liveSnapshotSAPIENTGraphEnv   = "SEMOPS_COP_SMOKE_SAPIENT_GRAPH_ENABLED"
	liveSnapshotKLVEnv            = "SEMOPS_COP_SMOKE_KLV_ENABLED"
	liveSnapshotWeatherEnv        = "SEMOPS_COP_SMOKE_WEATHER_ENABLED"
	liveSnapshotFusionEnv         = "SEMOPS_COP_SMOKE_FUSION_ENABLED"
	defaultExpectedTrackID        = "c360.edge-compose.cop.mavlink.track.system-42"
	defaultExpectedCoTTrack       = "c360.edge-compose.cop.tak.track.android-alpha"
	defaultExpectedFusionCoTTrack = "c360.edge-compose.cop.tak.track.android-fusion"
	defaultExpectedCoTTask        = "c360.edge-compose.cop.tak.task.marker-north-gate"
	defaultExpectedCoTChat        = "c360.edge-compose.cop.tak.advisory.chat-alpha-1"
	defaultExpectedHazard         = "c360.edge-compose.cop.cap.hazard_area.nws-demo-flood-warning"
	liveSnapshotPollTimeout       = 30 * time.Second
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
		} else if err := requireProductFeedBoundaryScenarioStatus(status); err != nil {
			lastErr = err
		} else if expectADSB && status.Summary.ADSBSnapshots < 2 {
			lastErr = fmt.Errorf("scenario runner ADS-B snapshots = %d, want at least 2",
				status.Summary.ADSBSnapshots)
		} else {
			requireHazard := status.Summary.CAPAlerts > 0
			expectedHazardID := firstNonEmpty(os.Getenv(liveSnapshotHazardEnv), defaultExpectedHazard)
			snapshot, err := fetchSnapshot(ctx, client, snapshotURL)
			if err != nil {
				lastErr = err
			} else if snapshotHasScenarioRunnerState(
				snapshot,
				expectedTrackID,
				expectedTaskID,
				expectedAdvisoryID,
				expectedHazardID,
				requireHazard,
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

func TestHostedCOPSnapshotReflectsSAPIENTGraphProvider(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	if snapshotURL == "" {
		t.Skipf("set %s to run the hosted COP SAPIENT graph snapshot smoke", liveSnapshotURLEnv)
	}
	expectSAPIENTGraph, err := boolFromEnv(liveSnapshotSAPIENTGraphEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !expectSAPIENTGraph {
		t.Skipf("set %s=true to run the hosted COP SAPIENT graph snapshot smoke", liveSnapshotSAPIENTGraphEnv)
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
		} else if snapshotHasSAPIENTTrack(snapshot) {
			return
		} else {
			lastErr = fmt.Errorf("snapshot missing SAPIENT graph track: scenario=%s tracks=%d",
				snapshot.Scenario, len(snapshot.Tracks))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect SAPIENT graph provider before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsHADRSharedAirspace(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	statusURL := os.Getenv(liveScenarioStatusEnv)
	if snapshotURL == "" || statusURL == "" {
		t.Skipf("set %s and %s to run the hosted COP shared-airspace smoke",
			liveSnapshotURLEnv, liveScenarioStatusEnv)
	}
	expectADSB, err := boolFromEnv(liveSnapshotADSBHTTPEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !expectADSB {
		t.Skipf("set %s=true to run the hosted COP shared-airspace smoke", liveSnapshotADSBHTTPEnv)
	}
	expectedTrackID := firstNonEmpty(os.Getenv(liveSnapshotTrackIDEnv), defaultExpectedTrackID)
	expectedTaskID := firstNonEmpty(os.Getenv(liveSnapshotCoTTaskEnv), defaultExpectedCoTTask)
	expectedAdvisoryID := firstNonEmpty(os.Getenv(liveSnapshotCoTChatEnv), defaultExpectedCoTChat)
	expectedHazardID := firstNonEmpty(os.Getenv(liveSnapshotHazardEnv), defaultExpectedHazard)
	expectCAP, err := boolFromEnv(liveSnapshotCAPHTTPEnv)
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
		} else if err := requireProductFeedBoundaryScenarioStatus(status); err != nil {
			lastErr = err
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
				expectCAP,
				false,
			) && snapshotHasADSBTrack(snapshot) {
				return
			} else {
				lastErr = fmt.Errorf("snapshot missing shared-airspace state: scenario=%s tracks=%d tasks=%d advisories=%d hazards=%d",
					snapshot.Scenario, len(snapshot.Tracks), len(snapshot.Tasks), len(snapshot.Advisories), len(snapshot.Hazards))
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect HADR shared airspace before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsKLVLocalMedia(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	if snapshotURL == "" {
		t.Skipf("set %s to run the hosted COP KLV snapshot smoke", liveSnapshotURLEnv)
	}
	expectKLV, err := boolFromEnv(liveSnapshotKLVEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !expectKLV {
		t.Skipf("set %s=true to run the hosted COP KLV snapshot smoke", liveSnapshotKLVEnv)
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
		} else if snapshotHasKLVSensorFootprint(snapshot) {
			return
		} else {
			lastErr = fmt.Errorf("snapshot missing KLV local-media sensor footprint: scenario=%s footprints=%d",
				snapshot.Scenario, len(snapshot.SensorFootprints))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect KLV local media before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsWeatherFixture(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	if snapshotURL == "" {
		t.Skipf("set %s to run the hosted COP weather snapshot smoke", liveSnapshotURLEnv)
	}
	expectWeather, err := boolFromEnv(liveSnapshotWeatherEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !expectWeather {
		t.Skipf("set %s=true to run the hosted COP weather snapshot smoke", liveSnapshotWeatherEnv)
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
		} else if snapshotHasWeatherObservation(snapshot) {
			return
		} else {
			lastErr = fmt.Errorf("snapshot missing weather fixture observations: scenario=%s observations=%d",
				snapshot.Scenario, len(snapshot.Weather))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect weather fixture before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPSnapshotReflectsFusionAssociation(t *testing.T) {
	snapshotURL := os.Getenv(liveSnapshotURLEnv)
	mavlinkAddr := os.Getenv(liveSnapshotUDPAddrEnv)
	cotAddr := os.Getenv(liveSnapshotCoTAddrEnv)
	if snapshotURL == "" || mavlinkAddr == "" || cotAddr == "" {
		t.Skipf("set %s, %s, and %s to run the hosted COP fusion association smoke",
			liveSnapshotURLEnv, liveSnapshotUDPAddrEnv, liveSnapshotCoTAddrEnv)
	}
	expectFusion, err := boolFromEnv(liveSnapshotFusionEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !expectFusion {
		t.Skipf("set %s=true to run the hosted COP fusion association smoke", liveSnapshotFusionEnv)
	}
	expectedMAVLinkTrackID := firstNonEmpty(os.Getenv(liveSnapshotTrackIDEnv), defaultExpectedTrackID)
	expectedCoTTrackID := firstNonEmpty(os.Getenv(liveSnapshotCoTTrackEnv), defaultExpectedFusionCoTTrack)

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	mavlinkFrames := generatedMAVLinkFrames(t)
	cotEvents := generatedFusionCoTEvents(t)
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	var reviewPosted bool
	for {
		if err := sendMAVLinkFrames(ctx, mavlinkAddr, mavlinkFrames); err != nil {
			lastErr = err
		}
		if err := sendCoTEvents(ctx, cotAddr, cotEvents); err != nil {
			lastErr = err
		}

		snapshot, err := fetchSnapshot(ctx, client, snapshotURL)
		if err != nil {
			lastErr = err
		} else if !snapshotHasTrack(snapshot, expectedMAVLinkTrackID) {
			lastErr = fmt.Errorf("snapshot missing MAVLink track %s before fusion association: tracks=%d associations=%d",
				expectedMAVLinkTrackID, len(snapshot.Tracks), len(snapshot.Associations))
		} else if !snapshotHasTrack(snapshot, expectedCoTTrackID) {
			lastErr = fmt.Errorf("snapshot missing close CoT track %s before fusion association: tracks=%d associations=%d",
				expectedCoTTrackID, len(snapshot.Tracks), len(snapshot.Associations))
		} else if association, ok := fusionAssociation(snapshot, expectedMAVLinkTrackID, expectedCoTTrackID); ok {
			if association.OperatorReview != nil &&
				association.OperatorReview.Decision == copapi.AssociationReviewChallenged &&
				association.OperatorReview.ReviewedBy == "smoke.operator" &&
				association.OperatorReview.AuthorityScope == copapi.DefaultAssociationReviewAuthorityScope {
				return
			}
			if !reviewPosted {
				if err := postAssociationReview(ctx, client, snapshotURL, association.ID); err != nil {
					lastErr = err
				} else {
					reviewPosted = true
					lastErr = fmt.Errorf("fusion association review posted; waiting for graph-backed readback")
				}
			} else {
				lastErr = fmt.Errorf("fusion association has no challenged operator review yet: association=%s", association.ID)
			}
		} else {
			lastErr = fmt.Errorf("snapshot missing fusion association between %s and %s: associations=%d",
				expectedMAVLinkTrackID, expectedCoTTrackID, len(snapshot.Associations))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP snapshot did not reflect fusion association before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPComponentPrometheusMetricsReflectFeedFlow(t *testing.T) {
	metricsURL := os.Getenv(liveComponentMetricsEnv)
	mavlinkAddr := os.Getenv(liveSnapshotUDPAddrEnv)
	cotAddr := os.Getenv(liveSnapshotCoTAddrEnv)
	if metricsURL == "" || mavlinkAddr == "" || cotAddr == "" {
		t.Skipf("set %s, %s, and %s to run the hosted component metrics smoke",
			liveComponentMetricsEnv, liveSnapshotUDPAddrEnv, liveSnapshotCoTAddrEnv)
	}
	expectADSB, err := boolFromEnv(liveSnapshotADSBHTTPEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectSAPIENT, err := boolFromEnv(liveSnapshotSAPIENTEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectSAPIENTGraph, err := boolFromEnv(liveSnapshotSAPIENTGraphEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectKLV, err := boolFromEnv(liveSnapshotKLVEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectWeather, err := boolFromEnv(liveSnapshotWeatherEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectFusion, err := boolFromEnv(liveSnapshotFusionEnv)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	mavlinkFrames := generatedMAVLinkFrames(t)
	cotEvents, err := cotcodec.MarshalEvents(cotcodec.SeedEvents(time.Now().UTC()))
	if err != nil {
		t.Fatalf("marshal cot seed events: %v", err)
	}
	if expectFusion {
		cotEvents = generatedFusionCoTEvents(t)
	}
	expected := []componentMetricExpectation{
		{Name: "semops-input-mavlink-udp", Feed: "mavlink", Role: "input"},
		{Name: "semops-processor-mavlink-decode", Feed: "mavlink", Role: "decoder"},
		{Name: "semops-processor-mavlink-project", Feed: "mavlink", Role: "projector"},
		{Name: "semops-input-cot-udp", Feed: "tak-cot", Role: "udp-input"},
		{Name: "semops-processor-cot-decode", Feed: "tak-cot", Role: "decoder"},
		{Name: "semops-processor-cot-project", Feed: "tak-cot", Role: "projector"},
	}
	if expectADSB {
		expected = append(expected,
			componentMetricExpectation{Name: "semops-input-adsb-http", Feed: "adsb", Role: "http-poller"},
			componentMetricExpectation{Name: "semops-processor-adsb-decode", Feed: "adsb", Role: "decoder"},
			componentMetricExpectation{Name: "semops-processor-adsb-project", Feed: "adsb", Role: "projector"},
		)
	}
	if expectSAPIENT {
		expected = append(expected,
			componentMetricExpectation{Name: "semops-input-sapient-http", Feed: "sapient", Role: "http-input"},
			componentMetricExpectation{Name: "semops-processor-sapient-decode", Feed: "sapient", Role: "decoder"},
		)
		if expectSAPIENTGraph {
			expected = append(expected,
				componentMetricExpectation{Name: "semops-processor-sapient-project", Feed: "sapient", Role: "projector"},
			)
		}
	}
	if expectKLV {
		expected = append(expected,
			componentMetricExpectation{Name: "semops-input-klv-media-ref", Feed: "klv", Role: "media-ref-input"},
			componentMetricExpectation{Name: "semops-processor-klv-demux", Feed: "klv", Role: "demux"},
			componentMetricExpectation{Name: "semops-processor-klv-decode", Feed: "klv", Role: "decoder"},
			componentMetricExpectation{Name: "semops-processor-klv-project", Feed: "klv", Role: "projector"},
		)
	}
	if expectWeather {
		expected = append(expected,
			componentMetricExpectation{Name: "semops-input-weather-fixture", Feed: "weather", Role: "fixture-input"},
			componentMetricExpectation{Name: "semops-processor-weather-decode", Feed: "weather", Role: "decoder"},
			componentMetricExpectation{Name: "semops-processor-weather-project", Feed: "weather", Role: "projector"},
		)
	}
	if expectFusion {
		expected = append(expected,
			componentMetricExpectation{Name: "semops-processor-fusion-candidates", Feed: "fusion", Role: "candidate-producer"},
			componentMetricExpectation{Name: "semops-processor-fusion-associate", Feed: "fusion", Role: "projector"},
		)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := sendMAVLinkFrames(ctx, mavlinkAddr, mavlinkFrames); err != nil {
			lastErr = err
		}
		if err := sendCoTEvents(ctx, cotAddr, cotEvents); err != nil {
			lastErr = err
		}

		metrics, err := fetchPrometheusMetrics(ctx, client, metricsURL)
		if err != nil {
			lastErr = err
		} else if missing := missingComponentFlow(metrics, expected); len(missing) == 0 {
			return
		} else {
			lastErr = fmt.Errorf("component metrics missing flow: %s", strings.Join(missing, ", "))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted component Prometheus metrics did not reflect feed flow before timeout: %v; last error: %v",
				ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func TestHostedCOPRuntimeReflectsFeedFlow(t *testing.T) {
	runtimeURL := os.Getenv(liveRuntimeURLEnv)
	mavlinkAddr := os.Getenv(liveSnapshotUDPAddrEnv)
	cotAddr := os.Getenv(liveSnapshotCoTAddrEnv)
	if runtimeURL == "" || mavlinkAddr == "" || cotAddr == "" {
		t.Skipf("set %s, %s, and %s to run the hosted runtime smoke",
			liveRuntimeURLEnv, liveSnapshotUDPAddrEnv, liveSnapshotCoTAddrEnv)
	}
	expectADSB, err := boolFromEnv(liveSnapshotADSBHTTPEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectSAPIENT, err := boolFromEnv(liveSnapshotSAPIENTEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectSAPIENTGraph, err := boolFromEnv(liveSnapshotSAPIENTGraphEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectKLV, err := boolFromEnv(liveSnapshotKLVEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectWeather, err := boolFromEnv(liveSnapshotWeatherEnv)
	if err != nil {
		t.Fatal(err)
	}
	expectFusion, err := boolFromEnv(liveSnapshotFusionEnv)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSnapshotPollTimeout)
	defer cancel()

	mavlinkFrames := generatedMAVLinkFrames(t)
	cotEvents, err := cotcodec.MarshalEvents(cotcodec.SeedEvents(time.Now().UTC()))
	if err != nil {
		t.Fatalf("marshal cot seed events: %v", err)
	}
	if expectFusion {
		cotEvents = generatedFusionCoTEvents(t)
	}
	expected := []runtimeFeedExpectation{
		{ID: "feed.mavlink", Healthy: 3, Total: 3, RequireFlow: true},
		{ID: "feed.tak", Healthy: 3, Total: 3, RequireFlow: true},
	}
	if expectADSB {
		expected = append(expected, runtimeFeedExpectation{ID: "feed.adsb", Healthy: 3, Total: 3, RequireFlow: true})
	}
	if expectSAPIENT {
		components := 2
		if expectSAPIENTGraph {
			components = 3
		}
		expected = append(expected, runtimeFeedExpectation{
			ID:          "feed.sapient",
			Healthy:     components,
			Total:       components,
			RequireFlow: true,
		})
	}
	if expectKLV {
		expected = append(expected, runtimeFeedExpectation{ID: "feed.klv", Healthy: 4, Total: 4, RequireFlow: true})
	}
	if expectWeather {
		expected = append(expected, runtimeFeedExpectation{ID: "feed.weather", Healthy: 3, Total: 3, RequireFlow: true})
	}
	if expectFusion {
		expected = append(expected, runtimeFeedExpectation{ID: "feed.fusion", Healthy: 2, Total: 2, RequireFlow: true})
	}

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := sendMAVLinkFrames(ctx, mavlinkAddr, mavlinkFrames); err != nil {
			lastErr = err
		}
		if err := sendCoTEvents(ctx, cotAddr, cotEvents); err != nil {
			lastErr = err
		}

		runtime, err := fetchRuntime(ctx, client, runtimeURL)
		if err != nil {
			lastErr = err
		} else if missing := missingRuntimeFeedFlow(runtime, expected); len(missing) == 0 {
			return
		} else {
			lastErr = fmt.Errorf("runtime missing feed flow: %s", strings.Join(missing, ", "))
		}

		select {
		case <-ctx.Done():
			t.Fatalf("hosted COP runtime did not reflect feed flow before timeout: %v; last error: %v",
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

func generatedFusionCoTEvents(t *testing.T) [][]byte {
	t.Helper()

	now := time.Now().UTC()
	raw, err := cotcodec.MarshalEvents([]cotcodec.Event{{
		UID:      "ANDROID-FUSION",
		Type:     cotcodec.TypeOperatorPosition,
		How:      cotcodec.DefaultHow,
		Time:     now,
		Start:    now,
		Stale:    now.Add(2 * time.Minute),
		Point:    &cotcodec.Point{Lat: 38.9001, Lon: -77.0002, HAE: 118.4, CE: 4, LE: 9},
		Callsign: "FUSION",
	}})
	if err != nil {
		t.Fatalf("marshal fusion cot event: %v", err)
	}
	return raw
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

func fetchRuntime(ctx context.Context, client *http.Client, runtimeURL string) (copapi.RuntimeSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, runtimeURL, nil)
	if err != nil {
		return copapi.RuntimeSnapshot{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return copapi.RuntimeSnapshot{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return copapi.RuntimeSnapshot{}, fmt.Errorf("runtime status = %d", resp.StatusCode)
	}
	var runtime copapi.RuntimeSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&runtime); err != nil {
		return copapi.RuntimeSnapshot{}, fmt.Errorf("decode runtime: %w", err)
	}
	return runtime, nil
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

func postAssociationReview(ctx context.Context, client *http.Client, snapshotURL string, associationID string) error {
	reviewURL := strings.TrimSuffix(snapshotURL, "/snapshot") +
		"/associations/" + url.PathEscape(associationID) + "/review"
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		reviewURL,
		strings.NewReader(`{"decision":"challenged","reviewed_by":"smoke.operator","comment":"smoke review"}`),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("association review status = %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
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
	IngressMode    string `json:"ingress_mode"`
	State          string `json:"state"`
	CompletedSteps int    `json:"completed_steps"`
	FailedSteps    int    `json:"failed_steps"`
	LastError      string `json:"last_error"`
	Summary        struct {
		CAPAlerts                     int `json:"cap_alerts"`
		ADSBSnapshots                 int `json:"adsb_snapshots"`
		FeedBoundaryDeliveries        int `json:"feed_boundary_deliveries"`
		ContractGraphMutationAttempts int `json:"contract_graph_mutation_attempts"`
		Mutations                     int `json:"mutations"`
	} `json:"summary"`
}

func requireProductFeedBoundaryScenarioStatus(status scenarioStatus) error {
	if status.IngressMode != "feed-boundary" {
		return fmt.Errorf("scenario runner ingress_mode = %q, want feed-boundary", status.IngressMode)
	}
	if status.CompletedSteps == 0 {
		return fmt.Errorf("scenario runner completed_steps = 0, want feed-boundary playback evidence")
	}
	if status.Summary.Mutations != 0 || status.Summary.ContractGraphMutationAttempts != 0 {
		return fmt.Errorf("scenario runner product status reported graph mutations: mutations=%d contract_graph_mutation_attempts=%d",
			status.Summary.Mutations,
			status.Summary.ContractGraphMutationAttempts)
	}
	if status.Summary.FeedBoundaryDeliveries != status.CompletedSteps {
		return fmt.Errorf("scenario runner feed-boundary deliveries = %d, want completed steps %d",
			status.Summary.FeedBoundaryDeliveries,
			status.CompletedSteps)
	}
	return nil
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
	requireHazard bool,
	expectADSB bool,
) bool {
	if !snapshotHasCoT(snapshot, expectedTrackID, expectedTaskID, expectedAdvisoryID) {
		return false
	}
	if requireHazard && !snapshotHasHazard(snapshot, expectedHazardID) {
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

func snapshotHasSAPIENTTrack(snapshot copapi.Snapshot) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	if !snapshotHasFeedStatus(snapshot, "feed.sapient", "live", "stale") {
		return false
	}
	for _, track := range snapshot.Tracks {
		if track.Source != "sapient" {
			continue
		}
		if track.Position.Lat == 0 || track.Position.Lon == 0 {
			return false
		}
		if track.Provenance.Owner != "semops.feed.sapient" || track.Provenance.SourceRef == "" {
			return false
		}
		return true
	}
	return false
}

func snapshotHasFeedStatus(snapshot copapi.Snapshot, feedID string, statuses ...string) bool {
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

func snapshotHasKLVSensorFootprint(snapshot copapi.Snapshot) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	for _, footprint := range snapshot.SensorFootprints {
		if footprint.Source != "klv" {
			continue
		}
		if footprint.SensorPosition.Lat == 0 || footprint.SensorPosition.Lon == 0 ||
			footprint.FrameCenter.Lat == 0 || footprint.FrameCenter.Lon == 0 {
			return false
		}
		if len(footprint.Ray) != 2 {
			return false
		}
		if footprint.MediaRef == "" || footprint.PacketRef == "" {
			return false
		}
		if footprint.PlatformDesignation == "" {
			return false
		}
		if footprint.Provenance.Owner != "semops.feed.klv" {
			return false
		}
		if !hasString(footprint.DecodedFields, "sensor_position") ||
			!hasString(footprint.DecodedFields, "frame_center") {
			return false
		}
		if !strings.Contains(footprint.ClaimPosture, "no STANAG conformance") {
			return false
		}
		return true
	}
	return false
}

func snapshotHasWeatherObservation(snapshot copapi.Snapshot) bool {
	if snapshot.Scenario != "phase-1-live-graph" {
		return false
	}
	var feedLive bool
	for _, feed := range snapshot.Feeds {
		if feed.ID == "feed.weather" && feed.Status == "live" {
			feedLive = true
			break
		}
	}
	if !feedLive {
		return false
	}
	for _, observation := range snapshot.Weather {
		if observation.Source != "weather" || observation.Provider == "" {
			continue
		}
		if observation.Variable == "" || observation.Unit == "" {
			return false
		}
		if observation.QueryShape == "" || observation.QueryGeometryWKT == "" || observation.Position == nil {
			return false
		}
		if observation.Provenance.Owner != "semops.feed.weather" || observation.Provenance.SourceRef == "" {
			return false
		}
		if !strings.Contains(observation.ClaimPosture, "no live-provider") {
			return false
		}
		return true
	}
	return false
}

func snapshotHasFusionAssociation(snapshot copapi.Snapshot, leftTrackID, rightTrackID string) bool {
	_, ok := fusionAssociation(snapshot, leftTrackID, rightTrackID)
	return ok
}

func fusionAssociation(snapshot copapi.Snapshot, leftTrackID, rightTrackID string) (copapi.Association, bool) {
	if snapshot.Scenario != "phase-1-live-graph" {
		return copapi.Association{}, false
	}
	for _, association := range snapshot.Associations {
		if association.Source != "fusion" {
			continue
		}
		if association.Provenance.Owner != "semops.fusion.structural" {
			continue
		}
		if !tracksMatchAssociation(association, leftTrackID, rightTrackID) {
			continue
		}
		if association.Algorithm == "" || association.Confidence <= 0 {
			return copapi.Association{}, false
		}
		if association.DistanceMeters == nil || *association.DistanceMeters > 250 {
			return copapi.Association{}, false
		}
		if association.TimeDeltaSeconds == nil || *association.TimeDeltaSeconds > 10 {
			return copapi.Association{}, false
		}
		if !strings.Contains(association.ClaimPosture, "no source-track merge") {
			return copapi.Association{}, false
		}
		if association.Status == "associated" || association.Status == "ambiguous" {
			return association, true
		}
		return copapi.Association{}, false
	}
	return copapi.Association{}, false
}

func tracksMatchAssociation(association copapi.Association, leftTrackID, rightTrackID string) bool {
	return (association.PrimaryTrackID == leftTrackID && association.CandidateTrackID == rightTrackID) ||
		(association.PrimaryTrackID == rightTrackID && association.CandidateTrackID == leftTrackID)
}

type componentMetricExpectation struct {
	Name string
	Feed string
	Role string
}

type runtimeFeedExpectation struct {
	ID          string
	Healthy     int
	Total       int
	RequireFlow bool
}

type prometheusSample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

type prometheusSnapshot []prometheusSample

func fetchPrometheusMetrics(ctx context.Context, client *http.Client, metricsURL string) (prometheusSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/plain")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics status = %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read metrics: %w", err)
	}
	return parsePrometheusMetrics(string(body))
}

func parsePrometheusMetrics(body string) (prometheusSnapshot, error) {
	var snapshot prometheusSnapshot
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		name, labels, err := parsePrometheusIdentity(fields[0])
		if err != nil {
			return nil, err
		}
		snapshot = append(snapshot, prometheusSample{Name: name, Labels: labels, Value: value})
	}
	return snapshot, nil
}

func parsePrometheusIdentity(identity string) (string, map[string]string, error) {
	open := strings.IndexByte(identity, '{')
	if open == -1 {
		return identity, nil, nil
	}
	if !strings.HasSuffix(identity, "}") {
		return "", nil, fmt.Errorf("metric labels missing closing brace")
	}
	labels := map[string]string{}
	for _, part := range strings.Split(identity[open+1:len(identity)-1], ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return "", nil, fmt.Errorf("metric label missing equals")
		}
		labels[key] = strings.Trim(value, `"`)
	}
	return identity[:open], labels, nil
}

func missingRuntimeFeedFlow(runtime copapi.RuntimeSnapshot, expected []runtimeFeedExpectation) []string {
	byID := map[string]copapi.RuntimeFeed{}
	for _, feed := range runtime.Feeds {
		byID[feed.ID] = feed
	}
	var missing []string
	for _, item := range expected {
		feed, ok := byID[item.ID]
		if !ok {
			missing = append(missing, item.ID+":missing")
			continue
		}
		if feed.Status != "flowing" {
			missing = append(missing, fmt.Sprintf("%s:status=%s", item.ID, feed.Status))
			continue
		}
		if feed.HealthyComponents < item.Healthy || feed.TotalComponents < item.Total {
			missing = append(missing, fmt.Sprintf("%s:health=%d/%d", item.ID, feed.HealthyComponents, feed.TotalComponents))
			continue
		}
		if item.RequireFlow && feed.MessagesPerSecond <= 0 {
			missing = append(missing, item.ID+":messages_per_second")
			continue
		}
		if item.RequireFlow && feed.LastActivity == nil {
			missing = append(missing, item.ID+":last_activity")
		}
	}
	return missing
}

func missingComponentFlow(snapshot prometheusSnapshot, expected []componentMetricExpectation) []string {
	var missing []string
	for _, item := range expected {
		labels := map[string]string{"component": item.Name, "feed": item.Feed, "role": item.Role}
		if snapshot.sum("semops_component_health_status", labels) <= 0 {
			missing = append(missing, item.Name+":health")
			continue
		}
		if snapshot.sum("semops_component_flow_messages_per_second", labels) <= 0 {
			missing = append(missing, item.Name+":messages_per_second")
			continue
		}
		if snapshot.sum("semops_component_flow_last_activity_timestamp_seconds", labels) <= 0 {
			missing = append(missing, item.Name+":last_activity")
		}
	}
	return missing
}

func (s prometheusSnapshot) sum(name string, labels map[string]string) float64 {
	var total float64
	for _, sample := range s {
		if sample.Name != name {
			continue
		}
		if prometheusLabelsMatch(sample.Labels, labels) {
			total += sample.Value
		}
	}
	return total
}

func prometheusLabelsMatch(got, want map[string]string) bool {
	for key, value := range want {
		if got[key] != value {
			return false
		}
	}
	return true
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

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
