package cap

import (
	"strings"
	"testing"
	"time"
)

func TestParseCAPAlertPreservesWarningAreasAndText(t *testing.T) {
	alert, err := Parse([]byte(sampleCAPAlert))
	if err != nil {
		t.Fatalf("parse cap alert: %v", err)
	}

	if alert.Identifier != "nws-demo-flood-warning" ||
		alert.Sender != "w-nws.webmaster@noaa.gov" ||
		alert.Status != "Actual" ||
		alert.MsgType != "Alert" ||
		alert.Scope != "Public" {
		t.Fatalf("alert = %+v", alert)
	}
	if alert.Sent != time.Date(2026, 6, 19, 15, 4, 5, 0, time.UTC) {
		t.Fatalf("sent = %s", alert.Sent)
	}

	info, ok := alert.PrimaryInfo()
	if !ok {
		t.Fatal("missing primary info")
	}
	if info.Event != "Flood Warning" ||
		info.Urgency != "Immediate" ||
		info.Severity != "Severe" ||
		info.Certainty != "Likely" {
		t.Fatalf("info = %+v", info)
	}
	if got := info.AdvisoryText(); !strings.Contains(got, "Flood Warning issued") ||
		!strings.Contains(got, "Move to higher ground") {
		t.Fatalf("advisory text = %q", got)
	}
	if len(info.Resources) != 1 || info.Resources[0].URI != "https://example.test/flood-detail" {
		t.Fatalf("resources = %+v", info.Resources)
	}
	if len(info.Areas) != 1 {
		t.Fatalf("areas = %+v", info.Areas)
	}
	area := info.Areas[0]
	if area.AreaDesc != "North Branch" {
		t.Fatalf("area desc = %q", area.AreaDesc)
	}
	if len(area.Polygons) != 1 || len(area.Polygons[0]) != 4 {
		t.Fatalf("polygons = %+v", area.Polygons)
	}
	if area.Polygons[0][0] != (Point{Lat: 38.895, Lon: -77.012}) {
		t.Fatalf("first polygon point = %+v", area.Polygons[0][0])
	}
	if len(area.Circles) != 1 || area.Circles[0].Center != (Point{Lat: 38.9, Lon: -77.01}) ||
		area.Circles[0].RadiusKM != 7.5 {
		t.Fatalf("circles = %+v", area.Circles)
	}
	if len(area.Geocodes) != 1 || area.Geocodes[0].Name != "SAME" || area.Geocodes[0].Value != "011001" {
		t.Fatalf("geocodes = %+v", area.Geocodes)
	}
}

func TestParseCAPCancelMessage(t *testing.T) {
	alert, err := Parse([]byte(strings.ReplaceAll(sampleCAPAlert, "<msgType>Alert</msgType>", "<msgType>Cancel</msgType>")))
	if err != nil {
		t.Fatalf("parse cancel: %v", err)
	}
	if alert.MsgType != "Cancel" {
		t.Fatalf("msgType = %q", alert.MsgType)
	}
}

func TestParseRejectsInvalidCAP(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "malformed xml",
			body: "<alert><identifier>x</identifier>",
			want: "EOF",
		},
		{
			name: "missing identifier",
			body: strings.Replace(sampleCAPAlert, "<identifier>nws-demo-flood-warning</identifier>", "", 1),
			want: "identifier",
		},
		{
			name: "invalid polygon",
			body: strings.Replace(sampleCAPAlert, "38.895,-77.012 38.907,-77.011 38.908,-76.992 38.896,-76.991", "38.895,-77.012 38.907,-77.011", 1),
			want: "polygon",
		},
		{
			name: "invalid circle radius",
			body: strings.Replace(sampleCAPAlert, "38.900,-77.010 7.5", "38.900,-77.010 nope", 1),
			want: "circle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.body))
			if err == nil {
				t.Fatal("expected parse error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

const sampleCAPAlert = `<?xml version="1.0" encoding="UTF-8"?>
<alert xmlns="urn:oasis:names:tc:emergency:cap:1.2">
  <identifier>nws-demo-flood-warning</identifier>
  <sender>w-nws.webmaster@noaa.gov</sender>
  <sent>2026-06-19T15:04:05Z</sent>
  <status>Actual</status>
  <msgType>Alert</msgType>
  <source>NWS</source>
  <scope>Public</scope>
  <info>
    <language>en-US</language>
    <category>Met</category>
    <event>Flood Warning</event>
    <urgency>Immediate</urgency>
    <severity>Severe</severity>
    <certainty>Likely</certainty>
    <effective>2026-06-19T15:04:05Z</effective>
    <expires>2026-06-19T18:04:05Z</expires>
    <senderName>National Weather Service</senderName>
    <headline>Flood Warning issued for North Branch</headline>
    <description>Flooding is occurring near low crossings.</description>
    <instruction>Move to higher ground. Avoid flooded roadways.</instruction>
    <web>https://example.test/flood</web>
    <resource>
      <resourceDesc>Flood detail</resourceDesc>
      <mimeType>text/html</mimeType>
      <uri>https://example.test/flood-detail</uri>
    </resource>
    <area>
      <areaDesc>North Branch</areaDesc>
      <polygon>38.895,-77.012 38.907,-77.011 38.908,-76.992 38.896,-76.991</polygon>
      <circle>38.900,-77.010 7.5</circle>
      <geocode>
        <valueName>SAME</valueName>
        <value>011001</value>
      </geocode>
    </area>
  </info>
</alert>`
