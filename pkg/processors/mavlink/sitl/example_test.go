//go:build ignore
// +build ignore

package sitl

import (
	"context"
	"fmt"
	"time"
)

// Example usage of the SITL controller
func ExampleController() {
	ctx := context.Background()
	config := DefaultConfig()

	// Create controller
	ctrl, err := NewController(ctx, config)
	if err != nil {
		fmt.Printf("Failed to create controller: %v\n", err)
		return
	}
	defer ctrl.Close()

	// Wait for connection
	fmt.Println("Waiting for connection...")
	time.Sleep(3 * time.Second)

	if !ctrl.IsConnected() {
		fmt.Println("Not connected to SITL")
		return
	}

	fmt.Println("Connected to SITL!")

	// Get status
	state := ctrl.GetState()
	fmt.Printf("Status: Mode=%s, Armed=%v, Battery=%d%%\n",
		state.FlightMode, state.Armed, state.BatteryPercent)

	// Example: Simple arm/disarm sequence
	if !ctrl.IsArmed() {
		fmt.Println("Arming drone...")
		if err := ctrl.Arm(); err != nil {
			fmt.Printf("Failed to arm: %v\n", err)
			return
		}

		if err := ctrl.WaitForArmed(ctx, true, 10*time.Second); err != nil {
			fmt.Printf("Failed to confirm armed: %v\n", err)
			return
		}

		fmt.Println("Armed successfully!")

		// Disarm after a moment
		time.Sleep(2 * time.Second)
		fmt.Println("Disarming...")

		if err := ctrl.Disarm(); err != nil {
			fmt.Printf("Failed to disarm: %v\n", err)
			return
		}

		if err := ctrl.WaitForArmed(ctx, false, 10*time.Second); err != nil {
			fmt.Printf("Failed to confirm disarmed: %v\n", err)
			return
		}

		fmt.Println("Disarmed successfully!")
	}
}

// Example basic flight scenario
func ExampleBasicFlight() {
	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	if err != nil {
		fmt.Printf("Failed to create controller: %v\n", err)
		return
	}
	defer ctrl.Close()

	// Wait for connection
	time.Sleep(3 * time.Second)
	if !ctrl.IsConnected() {
		fmt.Println("Not connected to SITL")
		return
	}

	// Perform basic flight
	fmt.Println("Starting basic flight...")
	if err := BasicFlight(ctx, ctrl, 10.0); err != nil {
		fmt.Printf("Basic flight failed: %v\n", err)
		return
	}

	fmt.Println("Basic flight completed!")
}

// Example status monitoring
func ExampleMonitorFlight() {
	ctx := context.Background()
	config := DefaultConfig()

	ctrl, err := NewController(ctx, config)
	if err != nil {
		fmt.Printf("Failed to create controller: %v\n", err)
		return
	}
	defer ctrl.Close()

	// Wait for telemetry
	time.Sleep(5 * time.Second)

	// Monitor for 30 seconds
	MonitorFlight(ctrl, 30*time.Second)
}
