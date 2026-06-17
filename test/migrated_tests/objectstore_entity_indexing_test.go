//go:build ignore
// +build ignore

package objectstore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c360/semstreams/processor/robotics/payloads"
	"github.com/c360/streamkit/cache"
	"github.com/c360/streamkit/natsclient"
)

// Helper function to create a store with default config for entity indexing tests
func createTestStoreForEntityIndexing(client *natsclient.Client, bucketName string) (*Store, error) {
	storeConfig := Config{
		BucketName: bucketName,
		DataCache:  cache.Config{Enabled: false},
	}
	return NewStoreWithConfig(context.Background(), client, storeConfig)
}

// TestEntityIndexing_GraphableMessage tests entity indexing with messages that implement Graphable
func TestEntityIndexing_GraphableMessage(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	store, err := createTestStoreForEntityIndexing(client, "TEST_ENTITY_MESSAGES")
	require.NoError(t, err)

	// Create a BatteryPayload which implements Graphable
	batteryMsg := payloads.NewBatteryPayload(42, 0, time.Now())
	batteryMsg.BatteryRemaining = 75

	// Store the message
	key, err := store.Store(context.Background(), batteryMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Verify entity metadata was stored
	entityIDs, err := store.GetEntityMetadata(context.Background(), key)
	require.NoError(t, err)
	assert.Len(t, entityIDs, 1, "Battery message should have one entity (triple authority)")

	// Battery can only assert facts about itself, not the drone
	// Using 6-part entity ID: org.platform.system.domain.type.instance
	expectedBatteryID := "c360.platform1.robotics.mav42.battery.0"
	assert.Contains(t, entityIDs, expectedBatteryID, "Should contain battery entity")
}

// TestEntityIndexing_QueryByEntityID tests querying messages by entity ID
func TestEntityIndexing_QueryByEntityID(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	store, err := createTestStoreForEntityIndexing(client, "TEST_ENTITY_QUERY")
	require.NoError(t, err)

	// Store multiple messages for the same drone
	droneID := uint8(123)
	baseTime := time.Now()
	battery1 := payloads.NewBatteryPayload(droneID, 0, baseTime)
	battery1.BatteryRemaining = 80

	battery2 := payloads.NewBatteryPayload(droneID, 1, baseTime.Add(2*time.Second))
	battery2.BatteryRemaining = 65

	// Store a message for different drone
	battery3 := payloads.NewBatteryPayload(99, 0, baseTime.Add(4*time.Second))
	battery3.BatteryRemaining = 50

	// Store all messages
	key1, err := store.Store(context.Background(), battery1)
	require.NoError(t, err)

	key2, err := store.Store(context.Background(), battery2)
	require.NoError(t, err)

	key3, err := store.Store(context.Background(), battery3)
	require.NoError(t, err)

	// Query messages for battery of drone 123
	// Using 6-part entity ID: org.platform.system.domain.type.instance
	targetEntityID := "c360.platform1.robotics.mav123.battery.0"
	foundKeys, err := store.ListByEntityID(context.Background(), targetEntityID)
	require.NoError(t, err)

	// NOTE: Due to key collision, only the last message for the same primary entity is stored
	// Both battery1 and battery2 have the same EntityID (c360.platform1.robotics.mav123.battery.0) and
	// are stored within the same time bucket, so battery2 overwrites battery1
	// This is expected behavior for last-writer-wins semantics
	assert.Len(t, foundKeys, 1, "Only one message should be stored due to key collision")

	// The stored message should be either key1 or key2 (whichever was stored last)
	// Due to the timing, it should be key2
	if len(foundKeys) > 0 {
		// Could be either key1 or key2 depending on timing, but typically key2 wins
		storedKey := foundKeys[0]
		assert.True(t, storedKey == key1 || storedKey == key2, "Should be one of the battery keys")
		assert.NotContains(t, foundKeys, key3, "Should not find messages for battery of drone 99")
	}

	// Query messages for battery of drone 99
	// Using 6-part entity ID: org.platform.system.domain.type.instance
	otherEntityID := "c360.platform1.robotics.mav99.battery.0"
	otherKeys, err := store.ListByEntityID(context.Background(), otherEntityID)
	require.NoError(t, err)

	assert.Len(t, otherKeys, 1)
	assert.Contains(t, otherKeys, key3)
}

// TestEntityIndexing_NonGraphableMessage tests backward compatibility with non-Graphable messages
func TestEntityIndexing_NonGraphableMessage(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	store, err := createTestStoreForEntityIndexing(client, "TEST_NON_GRAPHABLE")
	require.NoError(t, err)

	// Create a simple message that doesn't implement Graphable
	simpleMsg := map[string]any{
		"id":      "test-001",
		"content": "simple message",
		"value":   42,
	}

	// Store the message
	key, err := store.Store(context.Background(), simpleMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, key)

	// Non-Graphable messages should have no entity metadata
	entityIDs, err := store.GetEntityMetadata(context.Background(), key)
	require.NoError(t, err)
	assert.Empty(t, entityIDs, "Non-Graphable messages should have no entity metadata")

	// Should not be found in entity queries
	foundKeys, err := store.ListByEntityID(context.Background(), "any-entity-id")
	require.NoError(t, err)
	assert.NotContains(t, foundKeys, key)

	// But should still be retrievable normally
	data, err := store.Get(context.Background(), key)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

// TestEntityIndexing_MixedMessages tests a mix of Graphable and non-Graphable messages
func TestEntityIndexing_MixedMessages(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	store, err := createTestStoreForEntityIndexing(client, "TEST_MIXED_MESSAGES")
	require.NoError(t, err)

	// Store graphable message
	batteryMsg := payloads.NewBatteryPayload(1, 0, time.Now())
	batteryKey, err := store.Store(context.Background(), batteryMsg)
	require.NoError(t, err)

	// Store non-graphable message
	simpleMsg := map[string]string{"type": "simple", "data": "test"}
	simpleKey, err := store.Store(context.Background(), simpleMsg)
	require.NoError(t, err)

	// List all messages
	allKeys, err := store.List(context.Background(), "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allKeys), 2)

	// Query by entity ID should only return Graphable messages
	// Using 6-part entity ID: org.platform.system.domain.type.instance
	entityKeys, err := store.ListByEntityID(context.Background(), "c360.platform1.robotics.mav1.battery.0")
	require.NoError(t, err)
	assert.Len(t, entityKeys, 1)
	assert.Contains(t, entityKeys, batteryKey)
	assert.NotContains(t, entityKeys, simpleKey)

	// Both messages should be individually retrievable
	batteryData, err := store.Get(context.Background(), batteryKey)
	require.NoError(t, err)
	assert.NotEmpty(t, batteryData)

	simpleData, err := store.Get(context.Background(), simpleKey)
	require.NoError(t, err)
	assert.NotEmpty(t, simpleData)
}

// TestEntityIndexing_EmptyEntityQuery tests querying for non-existent entities
func TestEntityIndexing_EmptyEntityQuery(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	store, err := createTestStoreForEntityIndexing(client, "TEST_EMPTY_QUERY")
	require.NoError(t, err)

	// Store at least one message so the bucket has contents
	simpleMsg := map[string]any{"test": "data"}
	_, err = store.Store(context.Background(), simpleMsg)
	require.NoError(t, err)

	// Query for entity that doesn't exist
	keys, err := store.ListByEntityID(context.Background(), "nonexistent-entity")
	require.NoError(t, err)
	assert.Empty(t, keys, "Should return empty slice for non-existent entity")
}

// TestEntityIndexing_MetadataFormat tests the format of stored metadata
func TestEntityIndexing_MetadataFormat(t *testing.T) {
	client, cleanup := setupNATSContainer(t)
	defer cleanup()

	store, err := createTestStoreForEntityIndexing(client, "TEST_METADATA_FORMAT")
	require.NoError(t, err)

	// Create a message with known entity properties
	batteryMsg := payloads.NewBatteryPayload(42, 0, time.Now())
	key, err := store.Store(context.Background(), batteryMsg)
	require.NoError(t, err)

	// Get the ObjectInfo to examine headers directly
	info, err := store.store.GetInfo(context.Background(), key)
	require.NoError(t, err)

	// Verify specific headers exist and have expected format
	assert.Contains(t, info.Headers, "X-Entity-IDs")
	assert.Contains(t, info.Headers, "X-Entity-Types")
	assert.Contains(t, info.Headers, "X-Entity-Roles")
	assert.Contains(t, info.Headers, "X-Entity-Count")

	// Check entity ID format - battery messages only declare battery entity (triple authority)
	// Using 6-part entity ID: org.platform.system.domain.type.instance
	entityIDHeader := info.Headers["X-Entity-IDs"][0]
	assert.Equal(t, "c360.platform1.robotics.mav42.battery.0", entityIDHeader)

	// Check entity count - should be 1 (battery only, triple authority)
	countHeader := info.Headers["X-Entity-Count"][0]
	assert.Equal(t, "1", countHeader)

	// Check that entity types contains expected domain type
	typeHeader := info.Headers["X-Entity-Types"][0]
	assert.True(t, strings.Contains(typeHeader, "robotics"),
		"Entity type should contain 'robotics' domain")

	// Check that roles contain expected role
	roleHeader := info.Headers["X-Entity-Roles"][0]
	assert.True(t, strings.Contains(roleHeader, "primary"),
		"Entity role should contain 'primary' role")
}
