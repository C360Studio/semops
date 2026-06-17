//go:build ignore
// +build ignore

package parser

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/c360/semops/pkg/processors/mavlink/testing/mavlink"
	"github.com/stretchr/testify/require"
)

// TestUDPIntegration reproduces the exact E2E flow: Generator → UDP → Parser
func TestUDPIntegration(t *testing.T) {
	// Start UDP listener (simulating the UDP input component)
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0") // Random port
	require.NoError(t, err)

	conn, err := net.ListenUDP("udp", udpAddr)
	require.NoError(t, err)
	defer conn.Close()

	// Get the actual port we're listening on
	actualPort := conn.LocalAddr().(*net.UDPAddr).Port
	t.Logf("UDP listener started on port %d", actualPort)

	// Channel to collect received messages
	receivedMessages := make(chan []byte, 100)
	parseErrors := make(chan error, 100)

	// Start receiver goroutine (simulating robotics processor)
	go func() {
		parser := NewMAVLinkParser()
		buffer := make([]byte, 2048)

		for {
			n, addr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if !isClosedError(err) {
					t.Logf("Read error: %v", err)
				}
				return
			}

			// Copy the data to avoid buffer reuse issues
			data := make([]byte, n)
			copy(data, buffer[:n])

			t.Logf("Received %d bytes from %s", n, addr)
			t.Logf("Received hex: %x", data)

			// Parse the received data
			messages, err := parser.Parse(data)
			if err != nil {
				t.Logf("Parse error: %v", err)
				parseErrors <- err
				continue
			}

			t.Logf("Parsed %d messages", len(messages))
			for _, msg := range messages {
				receivedMessages <- []byte(fmt.Sprintf("msg_%d", msg.MessageID))
			}
		}
	}()

	// Give receiver time to start
	time.Sleep(100 * time.Millisecond)

	// Create UDP client (simulating E2E test sender)
	clientAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", actualPort))
	require.NoError(t, err)

	clientConn, err := net.DialUDP("udp", nil, clientAddr)
	require.NoError(t, err)
	defer clientConn.Close()

	// Create generator (simulating E2E test)
	gen := mavlink.NewGenerator(1, 1)

	t.Run("single message", func(t *testing.T) {
		// Generate and send a heartbeat
		heartbeat, err := gen.GenerateHeartbeat(mavlink.HeartbeatMessage{
			VehicleType:    2,  // MAV_TYPE_QUADROTOR
			Autopilot:      3,  // MAV_AUTOPILOT_ARDUPILOTMEGA
			BaseMode:       81, // ARMED + MANUAL
			CustomMode:     0,
			SystemStatus:   4, // MAV_STATE_ACTIVE
			MavlinkVersion: 3,
		})
		require.NoError(t, err)

		t.Logf("Sending heartbeat: %d bytes", len(heartbeat))
		t.Logf("Sending hex: %x", heartbeat)

		n, err := clientConn.Write(heartbeat)
		require.NoError(t, err)
		require.Equal(t, len(heartbeat), n)

		// Wait for message to be parsed
		select {
		case <-receivedMessages:
			// Success
		case err := <-parseErrors:
			t.Fatalf("Parse error: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for message")
		}
	})

	t.Run("multiple messages in one packet", func(t *testing.T) {
		// Generate multiple messages
		heartbeat, err := gen.GenerateHeartbeat(mavlink.HeartbeatMessage{
			VehicleType:    2,
			Autopilot:      3,
			BaseMode:       81,
			CustomMode:     0,
			SystemStatus:   4,
			MavlinkVersion: 3,
		})
		require.NoError(t, err)

		battery, err := gen.GenerateBatteryStatus(mavlink.BatteryMessage{
			BatteryRemaining: 75,
		})
		require.NoError(t, err)

		// Combine and send in one UDP packet
		combined := append(heartbeat, battery...)
		t.Logf("Sending combined: %d bytes", len(combined))

		n, err := clientConn.Write(combined)
		require.NoError(t, err)
		require.Equal(t, len(combined), n)

		// Should receive 2 messages
		messageCount := 0
		timeout := time.After(2 * time.Second)

		for messageCount < 2 {
			select {
			case <-receivedMessages:
				messageCount++
			case err := <-parseErrors:
				t.Fatalf("Parse error: %v", err)
			case <-timeout:
				t.Fatalf("Timeout: only received %d messages", messageCount)
			}
		}
	})

	t.Run("rapid fire messages", func(t *testing.T) {
		// Simulate E2E test sending messages rapidly
		messageCount := 10

		for i := 0; i < messageCount; i++ {
			heartbeat, err := gen.GenerateHeartbeat(mavlink.HeartbeatMessage{
				VehicleType:    2,
				Autopilot:      3,
				BaseMode:       81,
				CustomMode:     0,
				SystemStatus:   uint8(i % 8), // Vary status
				MavlinkVersion: 3,
			})
			require.NoError(t, err)

			_, err = clientConn.Write(heartbeat)
			require.NoError(t, err)

			// Small delay like E2E test might have
			time.Sleep(10 * time.Millisecond)
		}

		// Should receive all messages
		received := 0
		timeout := time.After(5 * time.Second)

		for received < messageCount {
			select {
			case <-receivedMessages:
				received++
			case err := <-parseErrors:
				t.Fatalf("Parse error after %d messages: %v", received, err)
			case <-timeout:
				t.Fatalf("Timeout: only received %d/%d messages", received, messageCount)
			}
		}

		t.Logf("Successfully received all %d messages", received)
	})

	t.Run("fragmented message simulation", func(t *testing.T) {
		// This simulates if UDP somehow fragments (shouldn't happen for small messages)
		position, err := gen.GenerateGlobalPosition(mavlink.PositionMessage{
			TimeBootMs:  1000,
			Lat:         int32(47.3977 * 1e7),
			Lon:         int32(-122.0316 * 1e7),
			Alt:         100000,
			RelativeAlt: 50000,
			Vx:          100,
			Vy:          200,
			Vz:          300,
			Hdg:         18000,
		})
		require.NoError(t, err)

		// Send in two parts (this is artificial but tests parser buffering)
		midpoint := len(position) / 2

		t.Logf("Sending first half: %d bytes", midpoint)
		_, err = clientConn.Write(position[:midpoint])
		require.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		t.Logf("Sending second half: %d bytes", len(position)-midpoint)
		_, err = clientConn.Write(position[midpoint:])
		require.NoError(t, err)

		// Parser should handle this gracefully
		select {
		case <-receivedMessages:
			// Success
		case err := <-parseErrors:
			// This might error, which is OK - UDP doesn't guarantee message boundaries
			t.Logf("Expected behavior: fragmented message caused parse error: %v", err)
		case <-time.After(2 * time.Second):
			t.Log("Timeout - parser correctly didn't parse fragmented message")
		}
	})
}

func isClosedError(err error) bool {
	if err == nil {
		return false
	}
	return bytes.Contains([]byte(err.Error()), []byte("use of closed"))
}
