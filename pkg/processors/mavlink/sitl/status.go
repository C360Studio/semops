package sitl

import (
	"context"
	"fmt"
	"time"
)

// Status query implementations

// GetPosition returns the current GPS position of the drone
func (c *Controller) GetPosition() (lat, lon, alt float64, err error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	// Check if we have recent position data
	if time.Since(c.state.LastPosition) > 5*time.Second {
		return 0, 0, 0, fmt.Errorf("no recent position data (last update: %v)", c.state.LastPosition)
	}
	
	return c.state.Latitude, c.state.Longitude, c.state.Altitude, nil
}

// GetBattery returns the current battery percentage
func (c *Controller) GetBattery() (float64, error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	// Check if we have recent battery data
	if time.Since(c.state.LastBattery) > 10*time.Second {
		return 0, fmt.Errorf("no recent battery data (last update: %v)", c.state.LastBattery)
	}
	
	return float64(c.state.BatteryPercent), nil
}

// GetMode returns the current flight mode
func (c *Controller) GetMode() (string, error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	if c.state.FlightMode == "" {
		return "", fmt.Errorf("flight mode not available")
	}
	
	return c.state.FlightMode, nil
}

// IsArmed returns true if the drone is armed
func (c *Controller) IsArmed() bool {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	return c.state.Armed
}

// GetAltitude returns the current altitude in meters MSL and relative to home
func (c *Controller) GetAltitude() (msl, relative float64, err error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	if time.Since(c.state.LastPosition) > 5*time.Second {
		return 0, 0, fmt.Errorf("no recent altitude data")
	}
	
	return c.state.Altitude, c.state.RelativeAlt, nil
}

// GetAttitude returns the current attitude (roll, pitch, yaw) in radians
func (c *Controller) GetAttitude() (roll, pitch, yaw float32, err error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	if time.Since(c.state.LastAttitude) > 5*time.Second {
		return 0, 0, 0, fmt.Errorf("no recent attitude data")
	}
	
	return c.state.Roll, c.state.Pitch, c.state.Yaw, nil
}

// GetVelocity returns the current velocity vector in m/s (NED frame)
func (c *Controller) GetVelocity() (vx, vy, vz float32, err error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	if time.Since(c.state.LastPosition) > 5*time.Second {
		return 0, 0, 0, fmt.Errorf("no recent velocity data")
	}
	
	return c.state.VelocityX, c.state.VelocityY, c.state.VelocityZ, nil
}

// GetSystemStatus returns the current system status
func (c *Controller) GetSystemStatus() (status uint8, statusName string, err error) {
	c.stateMux.RLock()
	defer c.stateMux.RUnlock()
	
	if !c.state.Connected {
		return 0, "", fmt.Errorf("not connected to drone")
	}
	
	return c.state.SystemStatus, c.getSystemStatusName(c.state.SystemStatus), nil
}

// Wait functions for synchronization

// WaitForArmed waits for the drone to be armed/disarmed with timeout
func (c *Controller) WaitForArmed(ctx context.Context, armed bool, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if c.IsArmed() == armed {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
	
	return fmt.Errorf("timeout waiting for armed=%v after %v", armed, timeout)
}

// WaitForMode waits for the drone to enter a specific flight mode with timeout
func (c *Controller) WaitForMode(ctx context.Context, mode string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if currentMode, err := c.GetMode(); err == nil && currentMode == mode {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
	
	return fmt.Errorf("timeout waiting for mode %s after %v", mode, timeout)
}

// WaitForAltitude waits for the drone to reach a specific altitude with tolerance
func (c *Controller) WaitForAltitude(ctx context.Context, targetAlt, tolerance float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		_, relativeAlt, err := c.GetAltitude()
		if err == nil {
			diff := relativeAlt - targetAlt
			if diff < 0 {
				diff = -diff
			}
			if diff <= tolerance {
				return nil
			}
		}
		
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
	
	return fmt.Errorf("timeout waiting for altitude %.1fm (tolerance %.1fm) after %v", targetAlt, tolerance, timeout)
}

// WaitForPosition waits for the drone to reach a specific GPS position with tolerance
func (c *Controller) WaitForPosition(ctx context.Context, targetLat, targetLon, tolerance float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		lat, lon, _, err := c.GetPosition()
		if err == nil {
			// Calculate distance using Haversine formula (approximate)
			dlat := targetLat - lat
			dlon := targetLon - lon
			distance := 111000.0 * (dlat*dlat + dlon*dlon*0.5) // rough approximation in meters
			
			if distance <= tolerance {
				return nil
			}
		}
		
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
	
	return fmt.Errorf("timeout waiting for position (%.6f, %.6f) tolerance %.1fm after %v", 
		targetLat, targetLon, tolerance, timeout)
}

// Helper functions

// getSystemStatusName converts system status code to human readable string
func (c *Controller) getSystemStatusName(status uint8) string {
	// Using constants from the robotics package
	switch status {
	case 0: // MAV_STATE_UNINIT
		return "Uninitialized"
	case 1: // MAV_STATE_BOOT
		return "Booting"
	case 2: // MAV_STATE_CALIBRATING
		return "Calibrating"
	case 3: // MAV_STATE_STANDBY
		return "Standby"
	case 4: // MAV_STATE_ACTIVE
		return "Active"
	case 5: // MAV_STATE_CRITICAL
		return "Critical"
	case 6: // MAV_STATE_EMERGENCY
		return "Emergency"
	case 7: // MAV_STATE_POWEROFF
		return "Power Off"
	case 8: // MAV_STATE_FLIGHT_TERMINATION
		return "Flight Termination"
	default:
		return fmt.Sprintf("Unknown (%d)", status)
	}
}