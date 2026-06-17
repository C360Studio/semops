//go:build ignore
// +build ignore

package objectstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/c360/semstreams/processor/robotics/payloads"
)

// ExampleStore demonstrates basic usage of the ObjectStore
func ExampleStore() {
	// Note: This example requires a running NATS server with JetStream
	// In real usage, you would get the JetStream context from your NATS client

	// For this example, we'll show the API usage without actually connecting
	// See store_test.go for full integration tests with test containers

	// Create a battery message (example from existing payload)
	batteryMsg := payloads.NewBatteryPayload(1, 0, time.Now())
	batteryMsg.BatteryRemaining = 85
	batteryMsg.Voltages = []uint16{4200, 4150, 4180, 4190} // mV per cell

	// Example of what the API would look like:
	// storeConfig := Config{BucketName: "MESSAGES", DataCache: cache.Config{Enabled: false}}
	// store, _ := NewStoreWithConfig(context.Background(), natsClient, storeConfig)

	// Store the message (would return key like: "robotics/battery/2024/01/19/14/battery_1_0_1705677443")
	// key, err := store.Store(batteryMsg)

	// Retrieve the message
	// data, err := store.Get(key)
	// var retrieved payloads.BatteryPayload
	// json.Unmarshal(data, &retrieved)

	// List messages by prefix
	// roboticsKeys, err := store.List("robotics/")
	// batteryKeys, err := store.List("robotics/battery/")

	fmt.Printf("Battery message entity ID: %s\n", batteryMsg.EntityID())
	fmt.Printf("Battery message type: %s\n", batteryMsg.Schema().String())

	// Output:
	// Battery message entity ID: c360.platform1.robotics.mav1.battery.0
	// Battery message type: robotics.battery.v1
}

// ExampleStore_keyGeneration demonstrates key generation patterns
func ExampleStore_keyGeneration() {
	// Mock store for key generation demo
	store := &Store{}

	// Different message types produce different key patterns
	messages := []any{
		&TestMessage{
			MessageType: "robotics.battery.v1",
			EntityName:  "drone_001",
		},
		&TestMessage{
			MessageType: "sensors.gps.v2",
			EntityName:  "gps_sensor_123",
		},
		map[string]string{"data": "raw message"}, // No interfaces
	}

	for _, msg := range messages {
		key := store.GenerateKey(msg)
		fmt.Printf("Key: %s\n", key)
	}

	// Output will be similar to:
	// Key: robotics/battery/2024/01/19/14/drone_001_1705677443
	// Key: sensors/gps/2024/01/19/14/gps_sensor_123_1705677443
	// Key: system/message/2024/01/19/14/unknown_1705677443
}

// ExampleStore_jsonStorage demonstrates JSON serialization
func ExampleStore_jsonStorage() {
	// Create a complex message with nested data
	complexMsg := &TestMessage{
		ID:          "example-001",
		MessageType: "robotics.mission.v1",
		EntityName:  "drone_mission_alpha",
		Content:     "Mission data",
		Timestamp:   time.Now(),
		Properties: map[string]any{
			"waypoints": []map[string]float64{
				{"lat": 37.7749, "lon": -122.4194, "alt": 100.0},
			},
			"priority": 1,
			"metadata": map[string]any{
				"created_by":   "pilot_001",
				"mission_type": "survey",
			},
		},
	}

	// This is what gets stored as JSON
	jsonData, _ := json.MarshalIndent(complexMsg, "", "  ")
	fmt.Printf("Stored JSON:\n%s\n", string(jsonData))

	// The key would be generated as:
	store := &Store{}
	key := store.GenerateKey(complexMsg)
	fmt.Printf("Storage key: %s\n", key)
}

// ExampleStore_entityIndexing demonstrates entity-based querying with Graphable messages
func ExampleStore_entityIndexing() {
	// Note: This example shows the entity indexing API
	// See integration tests for actual working examples with NATS

	fmt.Println("=== Entity Indexing Example ===")

	// When storing Graphable messages (like BatteryPayload):
	// 1. Message is stored as JSON with time-bucketed key
	// 2. Entity metadata is extracted and stored in NATS ObjectStore headers
	// 3. Can query messages by entity ID

	// Example BatteryPayload entities:
	// - "drone_42" (primary entity - the drone being monitored)
	// - "battery_42_0" (component entity - the specific battery)

	fmt.Println("Storing battery message...")
	fmt.Println("  Key: robotics/battery/2024/01/20/14/battery_42_0_1705754400")
	fmt.Println("  Entities extracted: [drone_42, battery_42_0]")
	fmt.Println("  Headers stored:")
	fmt.Println("    X-Entity-IDs: drone_42,battery_42_0")
	fmt.Println("    X-Entity-Types: robotics:Drone,robotics:Battery")
	fmt.Println("    X-Entity-Roles: primary,component")
	fmt.Println("    X-Entity-Count: 2")

	fmt.Println()
	fmt.Println("Querying by entity ID:")
	fmt.Println("  store.ListByEntityID('drone_42') -> [battery_key_1, position_key_2, ...]")
	fmt.Println("  store.GetEntityMetadata(key) -> ['drone_42', 'battery_42_0']")

	fmt.Println()
	fmt.Println("Backward compatibility:")
	fmt.Println("  Non-Graphable messages work unchanged - no entity metadata stored")
	fmt.Println("  All existing ObjectStore methods work as before")

	// Output:
	// === Entity Indexing Example ===
	// Storing battery message...
	//   Key: robotics/battery/2024/01/20/14/battery_42_0_1705754400
	//   Entities extracted: [drone_42, battery_42_0]
	//   Headers stored:
	//     X-Entity-IDs: drone_42,battery_42_0
	//     X-Entity-Types: robotics:Drone,robotics:Battery
	//     X-Entity-Roles: primary,component
	//     X-Entity-Count: 2
	//
	// Querying by entity ID:
	//   store.ListByEntityID('drone_42') -> [battery_key_1, position_key_2, ...]
	//   store.GetEntityMetadata(key) -> ['drone_42', 'battery_42_0']
	//
	// Backward compatibility:
	//   Non-Graphable messages work unchanged - no entity metadata stored
	//   All existing ObjectStore methods work as before
}
