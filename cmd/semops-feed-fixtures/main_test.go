package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
	capcodec "github.com/c360studio/semops/pkg/adapters/cap"
	sapientcodec "github.com/c360studio/semops/pkg/adapters/sapient"
)

func TestFixtureHandlerServesADSBAndSAPIENTFixtures(t *testing.T) {
	handler := fixtureHandler(func() time.Time {
		return time.Date(2026, 6, 21, 15, 0, 0, 0, time.UTC)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	adsbResp, err := http.Get(server.URL + "/adsb/states")
	if err != nil {
		t.Fatalf("get adsb fixture: %v", err)
	}
	defer adsbResp.Body.Close()
	if adsbResp.StatusCode != http.StatusOK {
		t.Fatalf("adsb fixture status = %d", adsbResp.StatusCode)
	}
	var raw struct {
		Time   int64             `json:"time"`
		States []json.RawMessage `json:"states"`
	}
	if err := json.NewDecoder(adsbResp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode adsb fixture: %v", err)
	}
	if raw.Time == 0 || len(raw.States) == 0 {
		t.Fatalf("adsb fixture = %+v", raw)
	}
	if _, err := adsbcodec.ParseOpenSkySnapshot(mustMarshalJSON(t, raw)); err != nil {
		t.Fatalf("parse adsb fixture: %v", err)
	}

	capResp, err := http.Get(server.URL + "/cap/alert")
	if err != nil {
		t.Fatalf("get cap fixture: %v", err)
	}
	defer capResp.Body.Close()
	if capResp.StatusCode != http.StatusOK {
		t.Fatalf("cap fixture status = %d", capResp.StatusCode)
	}
	capAlert, err := io.ReadAll(capResp.Body)
	if err != nil {
		t.Fatalf("read cap fixture: %v", err)
	}
	if _, err := capcodec.Parse(capAlert); err != nil {
		t.Fatalf("parse cap fixture: %v", err)
	}

	sapientResp, err := http.Get(server.URL + "/sapient/messages")
	if err != nil {
		t.Fatalf("get sapient fixture: %v", err)
	}
	defer sapientResp.Body.Close()
	if sapientResp.StatusCode != http.StatusOK {
		t.Fatalf("sapient fixture status = %d", sapientResp.StatusCode)
	}
	var sapientRaw json.RawMessage
	if err := json.NewDecoder(sapientResp.Body).Decode(&sapientRaw); err != nil {
		t.Fatalf("decode sapient fixture: %v", err)
	}
	msg, err := sapientcodec.ParseJSONMessage(sapientRaw)
	if err != nil {
		t.Fatalf("parse sapient fixture: %v", err)
	}
	if msg.Content != sapientcodec.ContentTaskAck {
		t.Fatalf("sapient fixture content = %s", msg.Content)
	}

	detectionResp, err := http.Get(server.URL + "/sapient/detections")
	if err != nil {
		t.Fatalf("get sapient detection fixture: %v", err)
	}
	defer detectionResp.Body.Close()
	if detectionResp.StatusCode != http.StatusOK {
		t.Fatalf("sapient detection fixture status = %d", detectionResp.StatusCode)
	}
	var detectionRaw json.RawMessage
	if err := json.NewDecoder(detectionResp.Body).Decode(&detectionRaw); err != nil {
		t.Fatalf("decode sapient detection fixture: %v", err)
	}
	detection, err := sapientcodec.ParseJSONMessage(detectionRaw)
	if err != nil {
		t.Fatalf("parse sapient detection fixture: %v", err)
	}
	if detection.Content != sapientcodec.ContentDetectionReport ||
		detection.DetectionReport == nil ||
		detection.DetectionReport.Location == nil {
		t.Fatalf("sapient detection fixture = %+v", detection)
	}
}

func TestFixtureAddrUsesDefaultAndEnv(t *testing.T) {
	if got := fixtureAddr(func(string) string { return "" }); got != defaultAddr {
		t.Fatalf("default addr = %q", got)
	}
	if got := fixtureAddr(func(string) string { return " :9000 " }); got != ":9000" {
		t.Fatalf("env addr = %q", got)
	}
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}
