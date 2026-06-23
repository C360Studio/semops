package sapient

import "testing"

func TestTaskAckFixtureJSONParses(t *testing.T) {
	msg, err := ParseJSONMessage(TaskAckFixtureJSON())
	if err != nil {
		t.Fatalf("parse task ack fixture: %v", err)
	}
	if msg.Content != ContentTaskAck || msg.TaskAck == nil {
		t.Fatalf("fixture content = %s task_ack=%v, want taskAck", msg.Content, msg.TaskAck)
	}
	if msg.NodeID != "a8654cdf-4328-47de-81fa-c495589e30c8" {
		t.Fatalf("fixture node id = %q", msg.NodeID)
	}
}

func TestDetectionFixtureJSONParses(t *testing.T) {
	msg, err := ParseJSONMessage(DetectionFixtureJSON())
	if err != nil {
		t.Fatalf("parse detection fixture: %v", err)
	}
	if msg.Content != ContentDetectionReport || msg.DetectionReport == nil {
		t.Fatalf("fixture content = %s detection=%v, want detectionReport", msg.Content, msg.DetectionReport)
	}
	if msg.DetectionReport.Location == nil ||
		msg.DetectionReport.Location.CoordinateSystem != "LOCATION_COORDINATE_SYSTEM_LAT_LNG_DEG_M" ||
		msg.DetectionReport.Location.Datum != "LOCATION_DATUM_WGS84_E" {
		t.Fatalf("detection location = %+v", msg.DetectionReport.Location)
	}
}
