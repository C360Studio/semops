//go:build ignore
// +build ignore

package sitl

import (
	"context"
	"fmt"
	"net"
	"time"

	mavlink "github.com/c360studio/semops/pkg/adapters/mavlink"
)

// Message handlers and processing

// registerDefaultHandlers registers default message handlers for telemetry
func (c *Controller) registerDefaultHandlers() {
	c.handlerMux.Lock()
	defer c.handlerMux.Unlock()

	c.messageHandlers[mavlink.MessageIDHeartbeat] = c.handleHeartbeat
	c.messageHandlers[mavlink.MessageIDGlobalPositionInt] = c.handleGlobalPosition
	c.messageHandlers[mavlink.MessageIDAttitude] = c.handleAttitude
	c.messageHandlers[mavlink.MessageIDBatteryStatus] = c.handleBatteryStatus
	c.messageHandlers[mavlink.MessageIDCommandAck] = c.handleCommandAck
}

// processMessages runs the main message processing loop
func (c *Controller) processMessages(ctx context.Context) {
	defer func() {
		select {
		case <-c.done:
		default:
			close(c.done)
		}
	}()

	buffer := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdown:
			return
		default:
			// Set read timeout to avoid blocking forever
			if err := c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				// Log but continue - read deadline is best effort
				continue
			}

			n, err := c.conn.Read(buffer)
			if err != nil {
				// Check if it's a timeout or actual error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Check shutdown signals
				}
				// Actual error, likely connection closed
				return
			}

			// Parse MAVLink packets
			packets, err := c.parser.Parse(buffer[:n])
			if err != nil {
				// Log error but continue processing
				continue
			}

			// Process each packet
			for _, packet := range packets {
				c.handleMessage(packet)
			}
		}
	}
}

// handleMessage dispatches messages to appropriate handlers
func (c *Controller) handleMessage(packet *mavlink.Packet) {
	c.handlerMux.RLock()
	handler, exists := c.messageHandlers[packet.MessageID]
	c.handlerMux.RUnlock()

	if exists {
		handler(packet)
	}
}

// handleHeartbeat processes HEARTBEAT messages
func (c *Controller) handleHeartbeat(packet *mavlink.Packet) {
	if packet.SystemID != c.targetSysID {
		return // Not from our target drone
	}

	c.stateMux.Lock()
	defer c.stateMux.Unlock()

	c.state.Connected = true
	c.state.LastHeartbeat = packet.Timestamp

	if packet.ParsedFields != nil {
		// Update armed status from base mode
		if baseMode, ok := packet.ParsedFields["base_mode"].(uint8); ok {
			c.state.Armed = (baseMode & mavlink.ModeFlagSafetyArmed) != 0
		}

		// Update system status
		if systemStatus, ok := packet.ParsedFields["system_status"].(uint8); ok {
			c.state.SystemStatus = systemStatus
		}

		// Update flight mode from custom mode
		if customMode, ok := packet.ParsedFields["custom_mode"].(uint32); ok {
			c.state.FlightMode = c.customModeToString(customMode)
		}
	}
}

// handleGlobalPosition processes GLOBAL_POSITION_INT messages
func (c *Controller) handleGlobalPosition(packet *mavlink.Packet) {
	if packet.SystemID != c.targetSysID {
		return
	}

	c.stateMux.Lock()
	defer c.stateMux.Unlock()

	if packet.ParsedFields != nil {
		// Update position
		if lat, ok := packet.ParsedFields["lat"].(int32); ok {
			c.state.Latitude = float64(lat) / 1e7
		}
		if lon, ok := packet.ParsedFields["lon"].(int32); ok {
			c.state.Longitude = float64(lon) / 1e7
		}
		if alt, ok := packet.ParsedFields["alt"].(int32); ok {
			c.state.Altitude = float64(alt) / 1000.0 // mm to meters
		}
		if relAlt, ok := packet.ParsedFields["relative_alt"].(int32); ok {
			c.state.RelativeAlt = float64(relAlt) / 1000.0 // mm to meters
		}

		// Update velocity
		if vx, ok := packet.ParsedFields["vx"].(int16); ok {
			c.state.VelocityX = float32(vx) / 100.0 // cm/s to m/s
		}
		if vy, ok := packet.ParsedFields["vy"].(int16); ok {
			c.state.VelocityY = float32(vy) / 100.0 // cm/s to m/s
		}
		if vz, ok := packet.ParsedFields["vz"].(int16); ok {
			c.state.VelocityZ = float32(vz) / 100.0 // cm/s to m/s
		}

		c.state.LastPosition = packet.Timestamp
	}
}

// handleAttitude processes ATTITUDE messages
func (c *Controller) handleAttitude(packet *mavlink.Packet) {
	if packet.SystemID != c.targetSysID {
		return
	}

	c.stateMux.Lock()
	defer c.stateMux.Unlock()

	if packet.ParsedFields != nil {
		if roll, ok := packet.ParsedFields["roll"].(float32); ok {
			c.state.Roll = roll
		}
		if pitch, ok := packet.ParsedFields["pitch"].(float32); ok {
			c.state.Pitch = pitch
		}
		if yaw, ok := packet.ParsedFields["yaw"].(float32); ok {
			c.state.Yaw = yaw
		}

		c.state.LastAttitude = packet.Timestamp
	}
}

// handleBatteryStatus processes BATTERY_STATUS messages
func (c *Controller) handleBatteryStatus(packet *mavlink.Packet) {
	if packet.SystemID != c.targetSysID {
		return
	}

	c.stateMux.Lock()
	defer c.stateMux.Unlock()

	if packet.ParsedFields != nil {
		if remaining, ok := packet.ParsedFields["battery_remaining"].(int8); ok {
			c.state.BatteryPercent = remaining
		}

		// Calculate average cell voltage
		if voltages, ok := packet.ParsedFields["voltages"].([]uint16); ok && len(voltages) > 0 {
			var total uint32
			var count int
			for _, voltage := range voltages {
				if voltage > 0 { // Only count non-zero voltages
					total += uint32(voltage)
					count++
				}
			}
			if count > 0 {
				c.state.BatteryVoltage = float32(total) / float32(count) / 1000.0 // mV to V
			}
		}

		c.state.LastBattery = packet.Timestamp
	}
}

// handleCommandAck processes COMMAND_ACK messages
func (c *Controller) handleCommandAck(packet *mavlink.Packet) {
	if packet.SystemID != c.targetSysID {
		return
	}

	if packet.ParsedFields == nil {
		return
	}

	command, ok1 := packet.ParsedFields["command"].(uint16)
	result, ok2 := packet.ParsedFields["result"].(uint8)

	if !ok1 || !ok2 {
		return
	}

	// Find waiting command acknowledgment
	c.ackMux.Lock()
	ackChan, exists := c.commandAcks[command]
	c.ackMux.Unlock()

	if !exists {
		return // No one waiting for this command
	}

	// Create result
	commandResult := CommandResult{
		Command: command,
		Result:  result,
		Success: result == 0, // MAV_RESULT_ACCEPTED = 0
		Message: c.mavResultToString(result),
	}

	// Send result to waiting goroutine (non-blocking)
	select {
	case ackChan <- commandResult:
	default:
		// Channel full or closed, ignore
	}
}

// sendHeartbeat sends periodic heartbeat messages
func (c *Controller) sendHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdown:
			return
		case <-ticker.C:
			// Send GCS heartbeat
			payload := c.encodeHeartbeat()
			frame, err := c.buildMAVLinkFrame(mavlink.MessageIDHeartbeat, payload)
			if err == nil {
				_ = c.sendFrame(frame) // Ignore error - heartbeat failure is not fatal
			}
		}
	}
}

// sendInitialHeartbeat sends an immediate heartbeat to establish communication
func (c *Controller) sendInitialHeartbeat() {
	payload := c.encodeHeartbeat()
	frame, err := c.buildMAVLinkFrame(mavlink.MessageIDHeartbeat, payload)
	if err == nil {
		_ = c.sendFrame(frame) // Ignore error - initial heartbeat failure is not fatal
	}
}

// encodeHeartbeat creates a GCS heartbeat payload
func (c *Controller) encodeHeartbeat() []byte {
	// HEARTBEAT payload: 9 bytes
	payload := make([]byte, 9)

	// custom_mode (uint32) - 0 for GCS
	// payload[0:4] already zero

	payload[4] = mavlink.TypeGCS          // type: Ground Control Station
	payload[5] = mavlink.AutopilotInvalid // autopilot: Invalid (GCS)
	payload[6] = 0                        // base_mode: no specific mode
	payload[7] = mavlink.StateActive      // system_status: Active
	payload[8] = mavlink.Version2         // mavlink_version

	return payload
}

// monitorConnection monitors connection health
func (c *Controller) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdown:
			return
		case <-ticker.C:
			c.stateMux.Lock()
			if time.Since(c.state.LastHeartbeat) > 10*time.Second {
				c.state.Connected = false
			}
			c.stateMux.Unlock()
		}
	}
}

// Helper conversion functions

// customModeToString converts ArduPilot custom mode to string
func (c *Controller) customModeToString(customMode uint32) string {
	modes := map[uint32]string{
		0:  "STABILIZE",
		1:  "ACRO",
		2:  "ALT_HOLD",
		3:  "AUTO",
		4:  "GUIDED",
		5:  "LOITER",
		6:  "RTL",
		7:  "CIRCLE",
		9:  "LAND",
		11: "DRIFT",
		13: "SPORT",
		14: "FLIP",
		15: "AUTOTUNE",
		16: "POSHOLD",
		17: "BRAKE",
		18: "THROW",
		19: "AVOID_ADSB",
		20: "GUIDED_NOGPS",
		21: "SMART_RTL",
		22: "FLOWHOLD",
		23: "FOLLOW",
		24: "ZIGZAG",
		25: "SYSTEMID",
		26: "AUTOROTATE",
	}

	if mode, exists := modes[customMode]; exists {
		return mode
	}

	return fmt.Sprintf("UNKNOWN(%d)", customMode)
}

// mavResultToString converts MAV_RESULT to human readable string
func (c *Controller) mavResultToString(result uint8) string {
	switch result {
	case 0:
		return "Command accepted"
	case 1:
		return "Command temporarily rejected"
	case 2:
		return "Command denied"
	case 3:
		return "Command unsupported"
	case 4:
		return "Command failed"
	case 5:
		return "Command in progress"
	default:
		return fmt.Sprintf("Unknown result (%d)", result)
	}
}
