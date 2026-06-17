//go:build ignore
// +build ignore

package robotics

import (
	"context"
	"testing"

	gonats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c360/semops/pkg/processors/mavlink/constants"
)

// mockNATSConnectionForMavlink creates a testcontainers NATS connection for mavlink testing
func mockNATSConnectionForMavlink(t *testing.T) *gonats.Conn {
	// Use shared connection from TestMain to avoid Docker resource exhaustion
	return getTestNATSConnection(t)
}

// TestMAVLinkParserIntegration verifies that the robotics processor correctly integrates the MAVLink parser
func TestMAVLinkParserIntegration(t *testing.T) {
	nc := mockNATSConnectionForMavlink(t)
	processor, err := NewRoboticsProcessor(nc)
	require.NoError(t, err)
	require.NotNil(t, processor)

	// Test processor can handle MAVLink data through the public API
	// This tests the behavior without accessing private fields
	testData := []byte{constants.MavlinkStxV1, 0x09, 0x00, 0x01, 0x01, 0x00} // Basic header
	err = processor.ProcessRawData(context.Background(), "input.udp.mavlink", testData)

	// The processor should handle data gracefully, even if packets are incomplete
	// We don't assert NoError here because incomplete packets may return errors,
	// but the processor should not crash
	assert.NotPanics(t, func() {
		processor.ProcessRawData(context.Background(), "input.udp.mavlink", testData)
	}, "Processor should handle incomplete MAVLink data without panicking")
}

// TestOutputSubjectMapping verifies the correct mapping of MAVLink message IDs to output subjects
func TestOutputSubjectMapping(t *testing.T) {
	nc := mockNATSConnectionForMavlink(t)
	processor, err := NewRoboticsProcessor(nc)
	require.NoError(t, err)

	// In the new architecture, all messages go to the same output port (ObjectStore)
	// This is configured via the port-based system
	expectedSubject := "storage.objectstore.write"

	tests := []struct {
		name      string
		messageID uint32
	}{
		{"heartbeat", constants.MavlinkMsgIdHeartbeat},
		{"position", constants.MavlinkMsgIdGlobalPositionInt},
		{"attitude", constants.MavlinkMsgIdAttitude},
		{"battery", constants.MavlinkMsgIdBatteryStatus},
		{"unknown", 999}, // Unknown message
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject := processor.getOutputSubject(tt.messageID)
			// All messages should go to the same output subject in the new architecture
			assert.Equal(t, expectedSubject, subject,
				"All messages should be routed to ObjectStore for semantic processing")
		})
	}
}

// TestMessageTypeNames verifies human-readable message type names
func TestMessageTypeNames(t *testing.T) {
	nc := mockNATSConnectionForMavlink(t)
	processor, err := NewRoboticsProcessor(nc)
	require.NoError(t, err)

	tests := []struct {
		messageID uint32
		expected  string
	}{
		{constants.MavlinkMsgIdHeartbeat, "HEARTBEAT"},
		{constants.MavlinkMsgIdGlobalPositionInt, "GLOBAL_POSITION_INT"},
		{constants.MavlinkMsgIdAttitude, "ATTITUDE"},
		{constants.MavlinkMsgIdBatteryStatus, "BATTERY_STATUS"},
		{999, "MESSAGE_999"}, // Unknown message
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			name := processor.getMessageTypeName(tt.messageID)
			assert.Equal(t, tt.expected, name)
		})
	}
}
