// Package rules provides battery monitoring rules for robotics systems
package rules

import (
	"testing"
	"time"

	gtypes "github.com/c360/semstreams/types/graph"
	message "github.com/c360/semstreams/message"
	roboticspayloads "github.com/c360/semops/pkg/processors/mavlink/payloads"
)

// TestBatteryMonitorRule_CreateEntityStates tests the graph entity creation functionality
func TestBatteryMonitorRule_CreateEntityStates(t *testing.T) {
	rule := NewBatteryMonitorRule("test-battery-monitor")
	messages := createTestBatteryMessages()

	// Test CreateEntityStates
	entityStates, err := rule.CreateEntityStates(messages)
	if err != nil {
		t.Fatalf("CreateEntityStates failed: %v", err)
	}

	// Should create 2 entities: drone and battery
	if len(entityStates) != 2 {
		t.Fatalf("Expected 2 entity states, got %d", len(entityStates))
	}

	// Find drone and battery entities
	droneState, batteryState := findEntityStates(t, entityStates)

	// Validate drone entity
	validateDroneEntity(t, droneState)

	// Validate battery entity
	validateBatteryEntity(t, batteryState)
}

// TestBatteryMonitorRule_CreateEntityStates_LowBattery tests critical battery status
func TestBatteryMonitorRule_CreateEntityStates_LowBattery(t *testing.T) {
	rule := NewBatteryMonitorRule("test-battery-monitor")

	// Create low battery payload
	battery := roboticspayloads.NewBatteryPayload(200, 0, time.Now())
	battery.BatteryRemaining = 15 // Below default 20% threshold

	mockMsg := message.NewBaseMessage(
		battery.Schema(),
		battery,
		"test-source",
	)

	entityStates, err := rule.CreateEntityStates([]message.Message{mockMsg})
	if err != nil {
		t.Fatalf("CreateEntityStates failed: %v", err)
	}

	// Find drone entity
	var droneState *gtypes.EntityState
	for _, state := range entityStates {
		if state.Node.Type == "robotics.drone" {
			droneState = state
		}
	}

	if droneState == nil {
		t.Fatal("Drone entity state not found")
	}

	// Should have warning status for low battery
	if droneState.Node.Status != gtypes.StatusWarning {
		t.Errorf("Expected drone status 'warning' for low battery, got '%s'", droneState.Node.Status)
	}
}

// TestBatteryMonitorRule_CreateEntityStates_Disabled tests disabled rule
func TestBatteryMonitorRule_CreateEntityStates_Disabled(t *testing.T) {
	rule := NewBatteryMonitorRule("test-battery-monitor")
	rule.Disable()

	battery := roboticspayloads.NewBatteryPayload(200, 0, time.Now())
	mockMsg := message.NewBaseMessage(
		battery.Schema(),
		battery,
		"test-source",
	)

	entityStates, err := rule.CreateEntityStates([]message.Message{mockMsg})
	if err != nil {
		t.Fatalf("CreateEntityStates failed: %v", err)
	}

	// Should return empty for disabled rule
	if len(entityStates) != 0 {
		t.Errorf("Expected 0 entity states for disabled rule, got %d", len(entityStates))
	}
}

// createTestBatteryMessages creates test messages for battery monitoring
func createTestBatteryMessages() []message.Message {
	battery := roboticspayloads.NewBatteryPayload(123, 1, time.Now())
	battery.BatteryRemaining = 75
	battery.Voltages = []uint16{3700, 3750, 3800}
	battery.CurrentBattery = 1500 // 15A in 10*mA units
	battery.Temperature = 2500    // 25°C in centigrade

	mockMsg := message.NewBaseMessage(
		battery.Schema(),
		battery,
		"test-source",
	)

	return []message.Message{mockMsg}
}

// findEntityStates finds drone and battery entities from the states list
func findEntityStates(t *testing.T, entityStates []*gtypes.EntityState) (*gtypes.EntityState, *gtypes.EntityState) {
	t.Helper()
	var droneState, batteryState *gtypes.EntityState
	for _, state := range entityStates {
		if state.Node.Type == "robotics.drone" {
			droneState = state
		} else if state.Node.Type == "robotics.battery" {
			batteryState = state
		}
	}

	if droneState == nil {
		t.Fatal("Drone entity state not found")
	}
	if batteryState == nil {
		t.Fatal("Battery entity state not found")
	}

	return droneState, batteryState
}

// validateDroneEntity validates drone entity properties and edges
func validateDroneEntity(t *testing.T, droneState *gtypes.EntityState) {
	t.Helper()
	if droneState.Node.ID != "drone_123" {
		t.Errorf("Expected drone ID 'drone_123', got '%s'", droneState.Node.ID)
	}
	if droneState.Node.Status != gtypes.StatusActive {
		t.Errorf("Expected drone status 'active', got '%s'", droneState.Node.Status)
	}

	// Check drone has POWERED_BY edge
	if len(droneState.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(droneState.Edges))
	}
	edge := droneState.Edges[0]
	if edge.EdgeType != "POWERED_BY" {
		t.Errorf("Expected edge type 'POWERED_BY', got '%s'", edge.EdgeType)
	}
	if edge.ToEntityID != "battery_123_1" {
		t.Errorf("Expected edge to 'battery_123_1', got '%s'", edge.ToEntityID)
	}
}

// validateBatteryEntity validates battery entity properties
func validateBatteryEntity(t *testing.T, batteryState *gtypes.EntityState) {
	t.Helper()
	if batteryState.Node.ID != "battery_123_1" {
		t.Errorf("Expected battery ID 'battery_123_1', got '%s'", batteryState.Node.ID)
	}
	if batteryState.Node.Status != gtypes.StatusActive {
		t.Errorf("Expected battery status 'active', got '%s'", batteryState.Node.Status)
	}

	// Check battery has no outgoing edges
	if len(batteryState.Edges) != 0 {
		t.Errorf("Expected battery to have 0 edges, got %d", len(batteryState.Edges))
	}

	// Check battery properties contain expected data
	props := batteryState.Node.Properties
	if charge, ok := props["charge_percent"].(int8); !ok || charge != 75 {
		t.Errorf("Expected charge_percent 75, got %v", props["charge_percent"])
	}
}