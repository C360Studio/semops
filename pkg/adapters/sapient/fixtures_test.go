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
