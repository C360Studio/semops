package cot

import (
	"strings"
	"testing"
	"time"
)

func TestUnmarshalSemLinkSeedEvents(t *testing.T) {
	now := "2026-06-19T13:20:00Z"
	tests := []struct {
		name      string
		raw       string
		wantUID   string
		wantType  string
		wantCall  string
		wantChat  string
		wantPoint *Point
	}{
		{
			name:     "operator dot",
			raw:      `<event version="2.0" uid="ANDROID-ALPHA" type="a-f-G-U-C" how="m-g" time="` + now + `"><point lat="38.8920" lon="-77.0350" hae="24" ce="5" le="5"/><detail><contact callsign="ALPHA"/></detail></event>`,
			wantUID:  "ANDROID-ALPHA",
			wantType: TypeOperatorPosition,
			wantCall: "ALPHA",
			wantPoint: &Point{
				Lat: 38.8920,
				Lon: -77.0350,
				HAE: 24,
				CE:  5,
				LE:  5,
			},
		},
		{
			name:      "checkpoint marker",
			raw:       `<event version="2.0" uid="MARKER-NORTH-GATE" type="u-d-p" how="m-g" time="` + now + `"><point lat="38.8940" lon="-77.0380" hae="0" ce="5" le="5"/><detail><contact callsign="North Gate"/><remarks>checkpoint</remarks></detail></event>`,
			wantUID:   "MARKER-NORTH-GATE",
			wantType:  TypeMarker,
			wantCall:  "North Gate",
			wantPoint: &Point{Lat: 38.8940, Lon: -77.0380, CE: 5, LE: 5},
		},
		{
			name:      "geochat",
			raw:       `<event version="2.0" uid="CHAT-ALPHA-1" type="b-t-f" how="h-g-i-g-o" time="` + now + `"><point lat="38.8920" lon="-77.0350" hae="24" ce="5" le="5"/><detail><contact callsign="ALPHA"/><remarks>hold at checkpoint</remarks><__chat senderUid="ANDROID-ALPHA" message="hold at checkpoint"/></detail></event>`,
			wantUID:   "CHAT-ALPHA-1",
			wantType:  TypeGeoChat,
			wantCall:  "ALPHA",
			wantChat:  "hold at checkpoint",
			wantPoint: &Point{Lat: 38.8920, Lon: -77.0350, HAE: 24, CE: 5, LE: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Unmarshal([]byte(tt.raw))
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got.UID != tt.wantUID || got.Type != tt.wantType || got.Callsign != tt.wantCall {
				t.Fatalf("event identity = (%q, %q, %q), want (%q, %q, %q)", got.UID, got.Type, got.Callsign, tt.wantUID, tt.wantType, tt.wantCall)
			}
			if got.Time.Format(time.RFC3339) != now {
				t.Fatalf("event time = %s, want %s", got.Time.Format(time.RFC3339), now)
			}
			if got.ChatText != tt.wantChat {
				t.Fatalf("chat text = %q, want %q", got.ChatText, tt.wantChat)
			}
			if got.Point == nil {
				t.Fatalf("point is nil")
			}
			if *got.Point != *tt.wantPoint {
				t.Fatalf("point = %+v, want %+v", *got.Point, *tt.wantPoint)
			}
		})
	}
}

func TestMarshalRoundTripTrack(t *testing.T) {
	now := time.Date(2026, 6, 19, 13, 21, 0, 0, time.UTC)
	raw, err := Marshal(Event{
		UID:       "UAS-42",
		Type:      TypeAirTrack,
		Time:      now,
		Point:     &Point{Lat: 38.9001, Lon: -77.0002, HAE: 118.4, CE: 4, LE: 9},
		Callsign:  "UAS 42",
		CourseDeg: 271.5,
		SpeedMPS:  17.2,
		HasTrack:  true,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(raw)
	for _, want := range []string{`uid="UAS-42"`, `type="a-f-A-M-F-Q"`, `<track course="271.5" speed="17.2"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("encoded CoT missing %q:\n%s", want, text)
		}
	}
	got, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal(Marshal()) error = %v", err)
	}
	if got.UID != "UAS-42" || !got.HasTrack || got.CourseDeg != 271.5 || got.SpeedMPS != 17.2 {
		t.Fatalf("round-tripped track = %+v", got)
	}
}

func TestUnmarshalGeoChatFallsBackToRemarks(t *testing.T) {
	raw := []byte(`<event version="2.0" uid="CHAT-BRAVO-1" type="b-t-f" how="h-g-i-g-o" time="2026-06-19T13:22:00.000Z"><detail><contact callsign="BRAVO"/><remarks>copy, holding west approach</remarks><__chat sender_uid="ANDROID-BRAVO"/></detail></event>`)

	got, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ChatText != "copy, holding west approach" || got.SenderUID != "ANDROID-BRAVO" {
		t.Fatalf("chat fallback = (%q, %q)", got.ChatText, got.SenderUID)
	}
}

func TestRejectInvalidCoT(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing uid", raw: `<event version="2.0" type="a-f-G-U-C" time="2026-06-19T13:23:00Z"></event>`},
		{name: "missing type", raw: `<event version="2.0" uid="ANDROID-ALPHA" time="2026-06-19T13:23:00Z"></event>`},
		{name: "bad time", raw: `<event version="2.0" uid="ANDROID-ALPHA" type="a-f-G-U-C" time="not-a-time"></event>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Unmarshal([]byte(tt.raw)); err == nil {
				t.Fatalf("Unmarshal() error = nil, want failure")
			}
		})
	}
}

func TestTypeClassifiers(t *testing.T) {
	if !IsOperatorType("a-h-G-U-C") || !IsMarkerType("b-m-p-s-p-i") || !IsGeoChatType(TypeGeoChat) || !IsAlertType(TypeAlert) {
		t.Fatalf("expected known CoT families to classify")
	}
	if IsAirTrackType(TypeOperatorPosition) || IsGeoChatType(TypeMarker) {
		t.Fatalf("unexpected cross-family classification")
	}
}
