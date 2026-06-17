//go:build ignore
// +build ignore

package sitl

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Flight scenario helpers for testing and demonstration

// Waypoint represents a GPS waypoint with altitude
type Waypoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"` // meters above home
}

// FlightPlan represents a sequence of waypoints with timing
type FlightPlan struct {
	Waypoints []Waypoint    `json:"waypoints"`
	HoldTime  time.Duration `json:"hold_time"` // Time to hold at each waypoint
	Tolerance float64       `json:"tolerance"` // Position tolerance in meters
	Timeout   time.Duration `json:"timeout"`   // Timeout for each waypoint
}

// BasicFlight performs a simple arm, takeoff, hold, land sequence
func BasicFlight(ctx context.Context, ctrl *Controller, altitude float64) error {
	fmt.Printf("Starting basic flight to %.1fm altitude\n", altitude)

	// Step 1: Arm the drone
	fmt.Print("Arming drone... ")
	if err := ctrl.Arm(); err != nil {
		return fmt.Errorf("failed to arm: %w", err)
	}
	fmt.Println("OK")

	// Wait for armed confirmation
	if err := ctrl.WaitForArmed(ctx, true, 10*time.Second); err != nil {
		return fmt.Errorf("failed to confirm armed: %w", err)
	}

	// Step 2: Take off
	fmt.Printf("Taking off to %.1fm... ", altitude)
	if err := ctrl.Takeoff(altitude); err != nil {
		return fmt.Errorf("failed to takeoff: %w", err)
	}
	fmt.Println("OK")

	// Wait to reach altitude (with 2m tolerance)
	if err := ctrl.WaitForAltitude(ctx, altitude, 2.0, 60*time.Second); err != nil {
		return fmt.Errorf("failed to reach altitude: %w", err)
	}
	fmt.Printf("Reached %.1fm altitude\n", altitude)

	// Step 3: Hold position for 5 seconds
	fmt.Print("Holding position for 5 seconds... ")
	time.Sleep(5 * time.Second)
	fmt.Println("OK")

	// Step 4: Land
	fmt.Print("Landing... ")
	if err := ctrl.Land(); err != nil {
		return fmt.Errorf("failed to land: %w", err)
	}
	fmt.Println("OK")

	// Wait for landing (altitude < 1m)
	if err := ctrl.WaitForAltitude(ctx, 1.0, 0.5, 60*time.Second); err != nil {
		fmt.Printf("Warning: landing timeout, but continuing: %v\n", err)
	}

	// Step 5: Disarm
	fmt.Print("Disarming... ")
	if err := ctrl.Disarm(); err != nil {
		return fmt.Errorf("failed to disarm: %w", err)
	}
	fmt.Println("OK")

	// Wait for disarm confirmation
	if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
		return fmt.Errorf("failed to confirm disarmed: %w", err)
	}

	fmt.Println("Basic flight completed successfully!")
	return nil
}

// SquarePattern flies a square pattern at the specified altitude
func SquarePattern(ctx context.Context, ctrl *Controller, centerLat, centerLon, altitude, sideLength float64) error {
	fmt.Printf("Starting square pattern flight (%.1fm sides at %.1fm altitude)\n", sideLength, altitude)

	// Calculate waypoints for square pattern (in degrees)
	degPerMeter := 1.0 / 111000.0 // Approximate conversion
	halfSide := sideLength * degPerMeter / 2

	waypoints := []Waypoint{
		{centerLat + halfSide, centerLon + halfSide, altitude}, // NE corner
		{centerLat + halfSide, centerLon - halfSide, altitude}, // NW corner
		{centerLat - halfSide, centerLon - halfSide, altitude}, // SW corner
		{centerLat - halfSide, centerLon + halfSide, altitude}, // SE corner
		{centerLat, centerLon, altitude},                       // Return to center
	}

	plan := FlightPlan{
		Waypoints: waypoints,
		HoldTime:  3 * time.Second,
		Tolerance: 5.0, // 5 meter tolerance
		Timeout:   60 * time.Second,
	}

	return Mission(ctx, ctrl, plan)
}

// CirclePattern flies a circular pattern at the specified altitude
func CirclePattern(ctx context.Context, ctrl *Controller, centerLat, centerLon, altitude, radius float64, numPoints int) error {
	fmt.Printf("Starting circle pattern flight (%.1fm radius, %d points at %.1fm altitude)\n",
		radius, numPoints, altitude)

	// Calculate waypoints for circle pattern
	degPerMeter := 1.0 / 111000.0 // Approximate conversion
	radiusDeg := radius * degPerMeter

	waypoints := make([]Waypoint, numPoints+1)

	for i := 0; i < numPoints; i++ {
		angle := float64(i) * 2.0 * math.Pi / float64(numPoints)
		lat := centerLat + radiusDeg*math.Cos(angle)
		lon := centerLon + radiusDeg*math.Sin(angle)
		waypoints[i] = Waypoint{lat, lon, altitude}
	}

	// Return to center
	waypoints[numPoints] = Waypoint{centerLat, centerLon, altitude}

	plan := FlightPlan{
		Waypoints: waypoints,
		HoldTime:  2 * time.Second,
		Tolerance: 5.0,
		Timeout:   60 * time.Second,
	}

	return Mission(ctx, ctrl, plan)
}

// Mission executes a flight plan with waypoints
func Mission(ctx context.Context, ctrl *Controller, plan FlightPlan) error {
	if len(plan.Waypoints) == 0 {
		return fmt.Errorf("flight plan has no waypoints")
	}

	fmt.Printf("Starting mission with %d waypoints\n", len(plan.Waypoints))

	// Get current position as starting point
	startLat, startLon, _, err := ctrl.GetPosition()
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	// Execute mission phases
	if err := prepareForMission(ctx, ctrl); err != nil {
		return err
	}

	firstAlt := plan.Waypoints[0].Altitude
	if err := takeoffToAltitude(ctx, ctrl, firstAlt); err != nil {
		return err
	}

	if err := executeWaypoints(ctx, ctrl, plan); err != nil {
		return err
	}

	if err := returnToStart(ctx, ctrl, startLat, startLon, firstAlt, plan.Tolerance, plan.Timeout); err != nil {
		return err
	}

	if err := completeMission(ctx, ctrl); err != nil {
		return err
	}

	fmt.Println("Mission completed successfully!")
	return nil
}

// prepareForMission arms the drone if needed
func prepareForMission(ctx context.Context, ctrl *Controller) error {
	if !ctrl.IsArmed() {
		fmt.Print("Arming drone... ")
		if err := ctrl.Arm(); err != nil {
			return fmt.Errorf("failed to arm: %w", err)
		}
		if err := ctrl.WaitForArmed(ctx, true, 10*time.Second); err != nil {
			return fmt.Errorf("failed to confirm armed: %w", err)
		}
		fmt.Println("OK")
	}
	return nil
}

// takeoffToAltitude takes off to the specified altitude
func takeoffToAltitude(ctx context.Context, ctrl *Controller, altitude float64) error {
	fmt.Printf("Taking off to %.1fm... ", altitude)
	if err := ctrl.Takeoff(altitude); err != nil {
		return fmt.Errorf("failed to takeoff: %w", err)
	}
	if err := ctrl.WaitForAltitude(ctx, altitude, 2.0, 60*time.Second); err != nil {
		return fmt.Errorf("failed to reach takeoff altitude: %w", err)
	}
	fmt.Println("OK")
	return nil
}

// executeWaypoints navigates through all waypoints in the plan
func executeWaypoints(ctx context.Context, ctrl *Controller, plan FlightPlan) error {
	for i, waypoint := range plan.Waypoints {
		if err := navigateToWaypoint(ctx, ctrl, i+1, waypoint, plan); err != nil {
			return err
		}
	}
	return nil
}

// navigateToWaypoint flies to a specific waypoint and holds if needed
func navigateToWaypoint(ctx context.Context, ctrl *Controller, waypointNum int, waypoint Waypoint, plan FlightPlan) error {
	fmt.Printf("Flying to waypoint %d (%.6f, %.6f, %.1fm)... ",
		waypointNum, waypoint.Latitude, waypoint.Longitude, waypoint.Altitude)

	// Navigate to waypoint
	if err := ctrl.Goto(waypoint.Latitude, waypoint.Longitude, waypoint.Altitude); err != nil {
		return fmt.Errorf("failed to send goto command for waypoint %d: %w", waypointNum, err)
	}

	// Wait to reach waypoint
	if err := ctrl.WaitForPosition(ctx, waypoint.Latitude, waypoint.Longitude, plan.Tolerance, plan.Timeout); err != nil {
		return fmt.Errorf("failed to reach waypoint %d: %w", waypointNum, err)
	}

	fmt.Println("OK")

	// Hold at waypoint
	if plan.HoldTime > 0 {
		fmt.Printf("Holding for %v... ", plan.HoldTime)
		time.Sleep(plan.HoldTime)
		fmt.Println("OK")
	}

	return nil
}

// returnToStart returns to the starting position
func returnToStart(ctx context.Context, ctrl *Controller, startLat, startLon, altitude, tolerance float64, timeout time.Duration) error {
	fmt.Printf("Returning to start position (%.6f, %.6f)... ", startLat, startLon)
	if err := ctrl.Goto(startLat, startLon, altitude); err != nil {
		return fmt.Errorf("failed to return to start: %w", err)
	}
	if err := ctrl.WaitForPosition(ctx, startLat, startLon, tolerance, timeout); err != nil {
		return fmt.Errorf("failed to reach start position: %w", err)
	}
	fmt.Println("OK")
	return nil
}

// completeMission lands and disarms the drone
func completeMission(ctx context.Context, ctrl *Controller) error {
	// Land
	fmt.Print("Landing... ")
	if err := ctrl.Land(); err != nil {
		return fmt.Errorf("failed to land: %w", err)
	}
	if err := ctrl.WaitForAltitude(ctx, 1.0, 0.5, 60*time.Second); err != nil {
		fmt.Printf("Warning: landing timeout: %v\n", err)
	}
	fmt.Println("OK")

	// Disarm
	fmt.Print("Disarming... ")
	if err := ctrl.Disarm(); err != nil {
		return fmt.Errorf("failed to disarm: %w", err)
	}
	if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
		return fmt.Errorf("failed to confirm disarmed: %w", err)
	}
	fmt.Println("OK")

	return nil
}

// EmergencyLand immediately lands the drone
func EmergencyLand(ctx context.Context, ctrl *Controller) error {
	fmt.Println("Emergency landing initiated!")

	// Switch to LAND mode
	if err := ctrl.SetMode("LAND"); err != nil {
		fmt.Printf("Warning: failed to set LAND mode: %v\n", err)
	}

	// Send land command
	if err := ctrl.Land(); err != nil {
		return fmt.Errorf("emergency land command failed: %w", err)
	}

	// Wait for landing
	if err := ctrl.WaitForAltitude(ctx, 1.0, 0.5, 120*time.Second); err != nil {
		return fmt.Errorf("emergency landing timeout: %w", err)
	}

	fmt.Println("Emergency landing completed")
	return nil
}

// ReturnToLaunch commands RTL and waits for completion
func ReturnToLaunch(ctx context.Context, ctrl *Controller) error {
	fmt.Println("Returning to launch...")

	if err := ctrl.RTL(); err != nil {
		return fmt.Errorf("RTL command failed: %w", err)
	}

	// Wait for mode change to RTL
	if err := ctrl.WaitForMode(ctx, "RTL", 10*time.Second); err != nil {
		fmt.Printf("Warning: RTL mode not confirmed: %v\n", err)
	}

	// Wait for altitude to decrease (indicating landing phase)
	time.Sleep(10 * time.Second) // Give time to fly home

	// Wait for final landing
	if err := ctrl.WaitForAltitude(ctx, 1.0, 0.5, 120*time.Second); err != nil {
		return fmt.Errorf("RTL landing timeout: %w", err)
	}

	fmt.Println("RTL completed successfully")
	return nil
}

// WaitForTakeoff is a helper that waits for takeoff to complete
func WaitForTakeoff(ctx context.Context, ctrl *Controller, targetAlt float64, timeout time.Duration) error {
	return ctrl.WaitForAltitude(ctx, targetAlt, 2.0, timeout)
}

// WaitForLanding is a helper that waits for landing to complete
func WaitForLanding(ctx context.Context, ctrl *Controller, timeout time.Duration) error {
	return ctrl.WaitForAltitude(ctx, 1.0, 0.5, timeout)
}

// MonitorFlight continuously monitors and reports flight status
func MonitorFlight(ctrl *Controller, duration time.Duration) {
	fmt.Printf("Monitoring flight for %v...\n", duration)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(start)
			if elapsed >= duration {
				fmt.Println("Monitoring complete")
				return
			}

			// Get status
			state := ctrl.GetState()

			fmt.Printf("[%02d:%02d] Armed: %v, Mode: %s, Alt: %.1fm, Battery: %d%%, Connected: %v\n",
				int(elapsed.Minutes()), int(elapsed.Seconds())%60,
				state.Armed, state.FlightMode, state.RelativeAlt, state.BatteryPercent, state.Connected)
		}
	}
}
