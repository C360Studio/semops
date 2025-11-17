package sitl

import (
	"fmt"
	"math"
	"time"

	"github.com/c360/semops/pkg/processors/mavlink/constants"
)

// Flight command implementations

// Arm arms the drone for flight
func (c *Controller) Arm() error {
	params := [7]float32{1, 0, 0, 0, 0, 0, 0} // param1=1 means arm
	result, err := c.sendCommand(constants.MAV_CMD_COMPONENT_ARM_DISARM, params, 5*time.Second)
	if err != nil {
		return fmt.Errorf("arm command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("arm command rejected: %s", result.Message)
	}
	
	return nil
}

// Disarm disarms the drone
func (c *Controller) Disarm() error {
	params := [7]float32{0, 0, 0, 0, 0, 0, 0} // param1=0 means disarm
	result, err := c.sendCommand(constants.MAV_CMD_COMPONENT_ARM_DISARM, params, 5*time.Second)
	if err != nil {
		return fmt.Errorf("disarm command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("disarm command rejected: %s", result.Message)
	}
	
	return nil
}

// Takeoff commands the drone to take off to the specified altitude
func (c *Controller) Takeoff(altitude float64) error {
	// MAV_CMD_NAV_TAKEOFF parameters:
	// param1: pitch (unused for multirotor)
	// param2: empty
	// param3: empty  
	// param4: yaw angle (NaN for current yaw)
	// param5: latitude (NaN for current position)
	// param6: longitude (NaN for current position)
	// param7: altitude in meters
	params := [7]float32{
		0,                    // pitch (unused for multirotor)
		0,                    // empty
		0,                    // empty
		float32(math.NaN()),  // yaw (current)
		float32(math.NaN()),  // lat (current)
		float32(math.NaN()),  // lon (current)
		float32(altitude),    // altitude
	}
	
	result, err := c.sendCommand(constants.MAV_CMD_NAV_TAKEOFF, params, 10*time.Second)
	if err != nil {
		return fmt.Errorf("takeoff command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("takeoff command rejected: %s", result.Message)
	}
	
	return nil
}

// Land commands the drone to land at its current position
func (c *Controller) Land() error {
	// MAV_CMD_NAV_LAND parameters:
	// param1: abort altitude (0 = no abort)
	// param2: precision land mode (0 = normal land)
	// param3: empty
	// param4: yaw angle (NaN for current yaw)
	// param5: latitude (NaN for current position)
	// param6: longitude (NaN for current position)
	// param7: altitude (0 for ground level)
	params := [7]float32{
		0,                    // abort altitude
		0,                    // precision land mode
		0,                    // empty
		float32(math.NaN()),  // yaw (current)
		float32(math.NaN()),  // lat (current)
		float32(math.NaN()),  // lon (current)
		0,                    // altitude (ground level)
	}
	
	result, err := c.sendCommand(constants.MAV_CMD_NAV_LAND, params, 10*time.Second)
	if err != nil {
		return fmt.Errorf("land command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("land command rejected: %s", result.Message)
	}
	
	return nil
}

// RTL commands the drone to return to launch position
func (c *Controller) RTL() error {
	// MAV_CMD_NAV_RETURN_TO_LAUNCH has no parameters
	params := [7]float32{0, 0, 0, 0, 0, 0, 0}
	
	result, err := c.sendCommand(constants.MAV_CMD_NAV_RETURN_TO_LAUNCH, params, 10*time.Second)
	if err != nil {
		return fmt.Errorf("RTL command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("RTL command rejected: %s", result.Message)
	}
	
	return nil
}

// SetMode changes the flight mode of the drone
func (c *Controller) SetMode(mode string) error {
	// Convert string mode to ArduPilot custom mode number
	customMode, err := c.stringToCustomMode(mode)
	if err != nil {
		return fmt.Errorf("invalid mode %s: %w", mode, err)
	}
	
	// MAV_CMD_DO_SET_MODE parameters:
	// param1: flight mode (1 = custom mode)
	// param2: custom mode number
	params := [7]float32{
		1,                      // mode flag (1 = custom mode)
		float32(customMode),    // custom mode number
		0, 0, 0, 0, 0,         // unused parameters
	}
	
	result, err := c.sendCommand(constants.MAV_CMD_DO_SET_MODE, params, 5*time.Second)
	if err != nil {
		return fmt.Errorf("set mode command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("set mode command rejected: %s", result.Message)
	}
	
	return nil
}

// Movement commands

// Goto commands the drone to fly to the specified GPS position  
func (c *Controller) Goto(lat, lon, alt float64) error {
	// MAV_CMD_NAV_WAYPOINT parameters:
	// param1: hold time at waypoint (seconds)
	// param2: acceptance radius (meters)
	// param3: pass radius (meters, 0 = no pass through)
	// param4: yaw angle (NaN for current yaw)
	// param5: latitude
	// param6: longitude  
	// param7: altitude
	params := [7]float32{
		0,                   // hold time
		5,                   // acceptance radius (5m)
		0,                   // pass radius
		float32(math.NaN()), // yaw (current)
		float32(lat),        // latitude
		float32(lon),        // longitude
		float32(alt),        // altitude
	}
	
	result, err := c.sendCommand(constants.MAV_CMD_NAV_WAYPOINT, params, 10*time.Second)
	if err != nil {
		return fmt.Errorf("goto command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("goto command rejected: %s", result.Message)
	}
	
	return nil
}

// SetVelocity sets the drone's velocity vector in m/s (NED frame)
func (c *Controller) SetVelocity(_, _, _ float64) error {
	// This requires sending SET_POSITION_TARGET_LOCAL_NED message
	// For now, return unsupported - would need separate message implementation
	return fmt.Errorf("SetVelocity not yet implemented - requires SET_POSITION_TARGET_LOCAL_NED message")
}

// SetYaw sets the drone's yaw angle in radians
func (c *Controller) SetYaw(yaw float64) error {
	// MAV_CMD_CONDITION_YAW parameters:
	// param1: target yaw angle (degrees)
	// param2: yaw speed (deg/s)
	// param3: direction (1 = clockwise, -1 = counter-clockwise)
	// param4: relative offset (1) or absolute angle (0)
	params := [7]float32{
		float32(yaw * 180.0 / math.Pi), // convert radians to degrees
		30,                             // yaw speed (30 deg/s)
		1,                              // clockwise
		0,                              // absolute angle
		0, 0, 0,                        // unused
	}
	
	result, err := c.sendCommand(constants.MAV_CMD_CONDITION_YAW, params, 5*time.Second)
	if err != nil {
		return fmt.Errorf("set yaw command failed: %w", err)
	}
	
	if !result.Success {
		return fmt.Errorf("set yaw command rejected: %s", result.Message)
	}
	
	return nil
}

// stringToCustomMode converts mode string to ArduPilot custom mode number
func (c *Controller) stringToCustomMode(mode string) (uint32, error) {
	// ArduCopter custom modes
	modes := map[string]uint32{
		"STABILIZE":    0,
		"ACRO":         1,
		"ALT_HOLD":     2,
		"AUTO":         3,
		"GUIDED":       4,
		"LOITER":       5,
		"RTL":          6,
		"CIRCLE":       7,
		"LAND":         9,
		"DRIFT":        11,
		"SPORT":        13,
		"FLIP":         14,
		"AUTOTUNE":     15,
		"POSHOLD":      16,
		"BRAKE":        17,
		"THROW":        18,
		"AVOID_ADSB":   19,
		"GUIDED_NOGPS": 20,
		"SMART_RTL":    21,
		"FLOWHOLD":     22,
		"FOLLOW":       23,
		"ZIGZAG":       24,
		"SYSTEMID":     25,
		"AUTOROTATE":   26,
	}
	
	if customMode, exists := modes[mode]; exists {
		return customMode, nil
	}
	
	return 0, fmt.Errorf("unknown mode: %s", mode)
}