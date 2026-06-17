//go:build ignore
// +build ignore

package objectstore

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c360/streamkit/cache"

	"github.com/c360/semstreams/processor/robotics/payloads"
)

// TestIntegration_BatteryPayload tests the ObjectStore with real BatteryPayload from the robotics package
func TestIntegration_BatteryPayload(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	storeConfig := Config{
		BucketName: "TEST_MESSAGES",
		DataCache:  cache.Config{Enabled: false},
	}
	store, err := NewStoreWithConfig(context.Background(), client, storeConfig)
	require.NoError(t, err)

	// Create a realistic battery payload
	batteryMsg := createTestBatteryPayload()

	// Store the battery message
	key, err := store.Store(context.Background(), batteryMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Debug: Print the actual key generated
	t.Logf("Generated key: %s", key)

	// Verify key format follows the expected pattern
	// Should be: robotics/battery/YYYY/MM/DD/HH/c360.platform1.robotics.mav42.battery.0_timestamp
	assert.True(t, strings.HasPrefix(key, "robotics/battery/"))
	assert.Contains(t, key, "c360.platform1.robotics.mav42.battery.0_")

	// Retrieve and verify message
	_ = verifyBatteryRetrieval(t, store, key, batteryMsg)

	// Test listing messages for this battery
	batteryKeys, err := store.List(context.Background(), "robotics/battery/")
	require.NoError(t, err)
	t.Logf("Found %d battery keys: %v", len(batteryKeys), batteryKeys)
	assert.GreaterOrEqual(t, len(batteryKeys), 1)

	// Verify our key is in the list
	found := false
	for _, listedKey := range batteryKeys {
		if listedKey == key {
			found = true
			break
		}
	}
	assert.True(t, found, "Stored key should be found in battery message list")
}

// TestIntegration_MultiplePayloads tests storing different payload types
func TestIntegration_MultiplePayloads(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	storeConfig := Config{
		BucketName: "TEST_MESSAGES",
		DataCache:  cache.Config{Enabled: false},
	}
	store, err := NewStoreWithConfig(context.Background(), client, storeConfig)
	require.NoError(t, err)

	// Store multiple different payload types
	batteryKey1, batteryKey2, heartbeatKey := storeMultiplePayloads(t, store)

	// Verify different key patterns
	assert.True(t, strings.HasPrefix(batteryKey1, "robotics/battery/"))
	assert.True(t, strings.HasPrefix(batteryKey2, "robotics/battery/"))
	assert.True(t, strings.HasPrefix(heartbeatKey, "robotics/heartbeat/"))

	assert.Contains(t, batteryKey1, "c360.platform1.robotics.mav1.battery.0_")
	assert.Contains(t, batteryKey2, "c360.platform1.robotics.mav2.battery.0_")
	assert.Contains(t, heartbeatKey, "c360.platform1.robotics.mav1.drone.0_")

	// List messages by domain
	roboticsKeys, err := store.List(context.Background(), "robotics/")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(roboticsKeys), 3)

	// List only battery messages
	batteryKeys, err := store.List(context.Background(), "robotics/battery/")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(batteryKeys), 2)

	// List only heartbeat messages
	heartbeatKeys, err := store.List(context.Background(), "robotics/heartbeat/")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(heartbeatKeys), 1)

	// Verify data integrity for each message type
	batteryData, err := store.Get(context.Background(), batteryKey1)
	require.NoError(t, err)
	var retrievedBattery payloads.BatteryPayload
	err = json.Unmarshal(batteryData, &retrievedBattery)
	require.NoError(t, err)
	assert.Equal(t, int8(80), retrievedBattery.BatteryRemaining)

	heartbeatData, err := store.Get(context.Background(), heartbeatKey)
	require.NoError(t, err)
	var retrievedHeartbeat payloads.HeartbeatPayload
	err = json.Unmarshal(heartbeatData, &retrievedHeartbeat)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), retrievedHeartbeat.SystemID)
}

// Helper functions for integration tests

func createTestBatteryPayload() *payloads.BatteryPayload {
	batteryMsg := payloads.NewBatteryPayload(42, 0, time.Now())
	batteryMsg.BatteryRemaining = 75
	batteryMsg.Voltages = []uint16{4150, 4120, 4180, 4160} // mV per cell
	batteryMsg.CurrentBattery = 250                        // 2.5A in 10*mA
	batteryMsg.Temperature = 2500                          // 25°C in centigrade
	batteryMsg.SetProperty("serial_number", "BAT12345")
	batteryMsg.SetProperty("manufacturer", "ACME Battery Co")
	return batteryMsg
}

func verifyBatteryRetrieval(t *testing.T, store *Store, key string, original *payloads.BatteryPayload) *payloads.BatteryPayload {
	// Retrieve the message
	data, err := store.Get(context.Background(), key)
	require.NoError(t, err)

	// Unmarshal back to BatteryPayload
	var retrieved payloads.BatteryPayload
	err = json.Unmarshal(data, &retrieved)
	require.NoError(t, err)

	// Verify the data integrity
	assert.Equal(t, original.SystemID, retrieved.SystemID)
	assert.Equal(t, original.BatteryID, retrieved.BatteryID)
	assert.Equal(t, original.BatteryRemaining, retrieved.BatteryRemaining)
	assert.Equal(t, original.Voltages, retrieved.Voltages)
	assert.Equal(t, original.CurrentBattery, retrieved.CurrentBattery)
	assert.Equal(t, original.Temperature, retrieved.Temperature)

	// Verify properties are preserved
	serialNumber, exists := retrieved.GetProperty("serial_number")
	assert.True(t, exists)
	assert.Equal(t, "BAT12345", serialNumber)

	manufacturer, exists := retrieved.GetProperty("manufacturer")
	assert.True(t, exists)
	assert.Equal(t, "ACME Battery Co", manufacturer)

	// Verify behavioral interfaces work
	assert.Equal(t, "c360.platform1.robotics.mav42.battery.0", retrieved.EntityID())
	assert.Equal(t, "robotics.battery.v1", retrieved.Schema().String())

	return &retrieved
}

func storeMultiplePayloads(t *testing.T, store *Store) (string, string, string) {
	// Store multiple different payload types
	battery1 := payloads.NewBatteryPayload(1, 0, time.Now())
	battery1.BatteryRemaining = 80

	battery2 := payloads.NewBatteryPayload(2, 0, time.Now())
	battery2.BatteryRemaining = 65

	heartbeat := payloads.NewHeartbeatPayload(1, 1, time.Now())

	// Store all messages
	batteryKey1, err := store.Store(context.Background(), battery1)
	require.NoError(t, err)

	batteryKey2, err := store.Store(context.Background(), battery2)
	require.NoError(t, err)

	heartbeatKey, err := store.Store(context.Background(), heartbeat)
	require.NoError(t, err)

	return batteryKey1, batteryKey2, heartbeatKey
}
