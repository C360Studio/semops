package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adsbcodec "github.com/c360studio/semops/pkg/adapters/adsb"
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
