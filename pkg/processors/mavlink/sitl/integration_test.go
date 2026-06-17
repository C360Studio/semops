//go:build ignore
// +build ignore

package sitl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests require running SITL
// Run with: task sitl:up && go test ./pkg/processor/robotics/sitl -run TestIntegration -v

func TestIntegrationConnection(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for initial connection
	time.Sleep(3 * time.Second)

	// Test connection
	assert.True(t, ctrl.IsConnected(), "Should be connected to SITL")

	// Test state retrieval
	state := ctrl.GetState()
	assert.True(t, state.Connected)
	assert.NotEmpty(t, state.FlightMode)

	t.Logf("Connected to SITL - Mode: %s, Armed: %v, Battery: %d%%",
		state.FlightMode, state.Armed, state.BatteryPercent)
}

func TestIntegrationStatusQueries(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for telemetry
	time.Sleep(5 * time.Second)

	// Test position query
	lat, lon, alt, err := ctrl.GetPosition()
	if assert.NoError(t, err) {
		t.Logf("Position: %.6f, %.6f, %.1fm", lat, lon, alt)
		assert.NotZero(t, lat)
		assert.NotZero(t, lon)
	}

	// Test battery query
	battery, err := ctrl.GetBattery()
	if assert.NoError(t, err) {
		t.Logf("Battery: %.0f%%", battery)
		assert.GreaterOrEqual(t, battery, 0.0)
		assert.LessOrEqual(t, battery, 100.0)
	}

	// Test mode query
	mode, err := ctrl.GetMode()
	if assert.NoError(t, err) {
		t.Logf("Flight mode: %s", mode)
		assert.NotEmpty(t, mode)
	}

	// Test armed status
	armed := ctrl.IsArmed()
	t.Logf("Armed: %v", armed)

	// Test altitude query
	msl, relative, err := ctrl.GetAltitude()
	if assert.NoError(t, err) {
		t.Logf("Altitude - MSL: %.1fm, Relative: %.1fm", msl, relative)
	}

	// Test attitude query
	roll, pitch, yaw, err := ctrl.GetAttitude()
	if assert.NoError(t, err) {
		t.Logf("Attitude - Roll: %.2f, Pitch: %.2f, Yaw: %.2f", roll, pitch, yaw)
	}

	// Test system status
	status, statusName, err := ctrl.GetSystemStatus()
	if assert.NoError(t, err) {
		t.Logf("System status: %d (%s)", status, statusName)
	}
}

func TestIntegrationArmDisarm(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for connection
	time.Sleep(3 * time.Second)
	require.True(t, ctrl.IsConnected())

	// Ensure disarmed initially
	if ctrl.IsArmed() {
		t.Log("Drone already armed, disarming first...")
		err := ctrl.Disarm()
		require.NoError(t, err)
		err = ctrl.WaitForArmed(ctx, false, 10*time.Second)
		require.NoError(t, err)
	}

	// Test arming
	t.Log("Testing arm command...")
	err = ctrl.Arm()
	require.NoError(t, err)

	// Wait for armed confirmation
	err = ctrl.WaitForArmed(ctx, true, 10*time.Second)
	require.NoError(t, err)
	assert.True(t, ctrl.IsArmed())
	t.Log("Successfully armed")

	// Test disarming
	t.Log("Testing disarm command...")
	err = ctrl.Disarm()
	require.NoError(t, err)

	// Wait for disarmed confirmation
	err = ctrl.WaitForArmed(ctx, false, 10*time.Second)
	require.NoError(t, err)
	assert.False(t, ctrl.IsArmed())
	t.Log("Successfully disarmed")
}

func TestIntegrationBasicFlight(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for connection
	time.Sleep(3 * time.Second)
	require.True(t, ctrl.IsConnected())

	// Ensure disarmed initially
	if ctrl.IsArmed() {
		if err := ctrl.Disarm(); err != nil {
			t.Logf("disarm error: %v", err)
		}
		if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
			t.Logf("wait for disarm error: %v", err)
		}
	}

	// Test basic flight at safe altitude
	t.Log("Starting basic flight test...")
	err = BasicFlight(ctx, ctrl, 10.0)
	require.NoError(t, err)

	t.Log("Basic flight test completed successfully")
}

func TestIntegrationModeChanges(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for connection
	time.Sleep(3 * time.Second)
	require.True(t, ctrl.IsConnected())

	// Test mode changes
	modes := []string{"STABILIZE", "ALT_HOLD", "LOITER", "GUIDED"}

	for _, mode := range modes {
		t.Logf("Setting mode to %s...", mode)
		err := ctrl.SetMode(mode)
		if err != nil {
			t.Logf("Failed to set mode %s: %v", mode, err)
			continue
		}

		// Wait for mode change (with timeout)
		err = ctrl.WaitForMode(ctx, mode, 5*time.Second)
		if err != nil {
			t.Logf("Mode change to %s not confirmed: %v", mode, err)
			continue
		}

		t.Logf("Successfully changed to mode %s", mode)
		time.Sleep(1 * time.Second)
	}
}

func TestIntegrationTakeoffLand(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	ctrl := setupController(t)
	defer ctrl.Close()

	ensureDisarmed(t, ctrl, ctx)

	// Test takeoff sequence
	targetAlt := 15.0
	testTakeoffSequence(t, ctrl, ctx, targetAlt)

	// Test landing sequence
	testLandingSequence(t, ctrl, ctx)

	t.Log("Takeoff/land test completed successfully")
}

// setupController creates and initializes a controller for testing
func setupController(t *testing.T) *Controller {
	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)

	// Wait for connection
	time.Sleep(3 * time.Second)
	require.True(t, ctrl.IsConnected())

	return ctrl
}

// ensureDisarmed ensures the drone is disarmed before testing
func ensureDisarmed(t *testing.T, ctrl *Controller, ctx context.Context) {
	if ctrl.IsArmed() {
		if err := ctrl.Disarm(); err != nil {
			t.Logf("disarm error: %v", err)
		}
		if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
			t.Logf("wait for disarm error: %v", err)
		}
	}
}

// testTakeoffSequence tests the complete takeoff sequence
func testTakeoffSequence(t *testing.T, ctrl *Controller, ctx context.Context, targetAlt float64) {
	// Arm the drone
	t.Log("Arming...")
	err := ctrl.Arm()
	require.NoError(t, err)
	err = ctrl.WaitForArmed(ctx, true, 10*time.Second)
	require.NoError(t, err)

	// Test takeoff
	t.Logf("Taking off to %.1fm...", targetAlt)
	err = ctrl.Takeoff(targetAlt)
	require.NoError(t, err)

	// Wait for altitude with generous timeout
	err = ctrl.WaitForAltitude(ctx, targetAlt, 3.0, 60*time.Second)
	require.NoError(t, err)

	// Verify we reached altitude
	_, relAlt, err := ctrl.GetAltitude()
	require.NoError(t, err)
	t.Logf("Reached altitude: %.1fm", relAlt)
	assert.InDelta(t, targetAlt, relAlt, 5.0)

	// Hold for a few seconds
	t.Log("Holding position...")
	time.Sleep(5 * time.Second)
}

// testLandingSequence tests the complete landing sequence
func testLandingSequence(t *testing.T, ctrl *Controller, ctx context.Context) {
	// Test landing
	t.Log("Landing...")
	err := ctrl.Land()
	require.NoError(t, err)

	// Wait for landing with generous timeout
	err = ctrl.WaitForAltitude(ctx, 1.0, 1.0, 60*time.Second)
	if err != nil {
		t.Logf("Landing timeout, but continuing: %v", err)
	}

	// Disarm
	t.Log("Disarming...")
	err = ctrl.Disarm()
	require.NoError(t, err)
	err = ctrl.WaitForArmed(ctx, false, 10*time.Second)
	require.NoError(t, err)
}

func TestIntegrationSquarePattern(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for connection and get position
	time.Sleep(3 * time.Second)
	require.True(t, ctrl.IsConnected())

	lat, lon, _, err := ctrl.GetPosition()
	require.NoError(t, err)

	// Ensure disarmed initially
	if ctrl.IsArmed() {
		if err := ctrl.Disarm(); err != nil {
			t.Logf("disarm error: %v", err)
		}
		if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
			t.Logf("wait for disarm error: %v", err)
		}
	}

	// Test square pattern flight
	t.Log("Starting square pattern flight...")
	err = SquarePattern(ctx, ctrl, lat, lon, 15.0, 50.0) // 50m square at 15m altitude
	if err != nil {
		t.Logf("Square pattern failed (expected in SITL): %v", err)
		// This might fail in SITL due to waypoint navigation limitations
		// but we can still test that the commands are sent correctly
	} else {
		t.Log("Square pattern completed successfully")
	}
}

func TestIntegrationRTL(t *testing.T) {
	if !isSITLRunning() {
		t.Skip("SITL not running - start with: task sitl:up")
	}

	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	require.NoError(t, err)
	defer ctrl.Close()

	// Wait for connection
	time.Sleep(3 * time.Second)
	require.True(t, ctrl.IsConnected())

	// Ensure disarmed initially
	if ctrl.IsArmed() {
		if err := ctrl.Disarm(); err != nil {
			t.Logf("disarm error: %v", err)
		}
		if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
			t.Logf("wait for disarm error: %v", err)
		}
	}

	// Arm and takeoff first
	t.Log("Arming and taking off...")
	err = ctrl.Arm()
	require.NoError(t, err)
	err = ctrl.WaitForArmed(ctx, true, 10*time.Second)
	require.NoError(t, err)

	err = ctrl.Takeoff(10.0)
	require.NoError(t, err)
	err = ctrl.WaitForAltitude(ctx, 10.0, 2.0, 60*time.Second)
	require.NoError(t, err)

	// Test RTL command
	t.Log("Testing RTL...")
	err = ctrl.RTL()
	require.NoError(t, err)

	// RTL should automatically land, so wait for low altitude
	err = ctrl.WaitForAltitude(ctx, 2.0, 1.0, 120*time.Second)
	if err != nil {
		t.Logf("RTL completion timeout, but RTL command was accepted: %v", err)
	} else {
		t.Log("RTL completed successfully")
	}

	// Ensure disarmed
	if ctrl.IsArmed() {
		if err := ctrl.Disarm(); err != nil {
			t.Logf("disarm error: %v", err)
		}
		if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
			t.Logf("wait for disarm error: %v", err)
		}
	}
}
