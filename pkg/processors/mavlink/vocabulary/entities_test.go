package vocabulary

import (
	"testing"

	"github.com/c360/semstreams/message"
	"github.com/stretchr/testify/assert"
)

func TestEntityTypeIRIs(t *testing.T) {
	// Test that all entity types have corresponding IRIs
	expectedMappings := map[string]string{
		EntityTypeDrone.Key():          "https://semstreams.c360.io/robotics#Drone",
		EntityTypeMission.Key():        "https://semstreams.c360.io/robotics#Mission",
		EntityTypeOperator.Key():       "https://semstreams.c360.io/robotics#Operator",
		EntityTypeFormation.Key():      "https://semstreams.c360.io/robotics#Formation",
		EntityTypeBattery.Key():        "https://semstreams.c360.io/robotics#Battery",
		EntityTypePosition.Key():       "https://semstreams.c360.io/robotics#Position",
		EntityTypeAttitude.Key():       "https://semstreams.c360.io/robotics#Attitude",
		EntityTypeCriticalStatus.Key(): "https://semstreams.c360.io/robotics#CriticalStatus",
	}

	for entityTypeKey, expectedIRI := range expectedMappings {
		t.Run(entityTypeKey, func(t *testing.T) {
			iri, exists := EntityTypeIRIs[entityTypeKey]
			assert.True(t, exists, "EntityTypeIRIs should contain mapping for %s", entityTypeKey)
			assert.Equal(t, expectedIRI, iri, "IRI for %s should match expected value", entityTypeKey)
		})
	}
}

func TestGetEntityTypeIRI(t *testing.T) {
	tests := []struct {
		name       string
		entityType message.EntityType
		expected   string
	}{
		{
			name:       "existing drone type",
			entityType: EntityTypeDrone,
			expected:   "https://semstreams.c360.io/robotics#Drone",
		},
		{
			name:       "existing battery type", 
			entityType: EntityTypeBattery,
			expected:   "https://semstreams.c360.io/robotics#Battery",
		},
		{
			name:       "existing mission type",
			entityType: EntityTypeMission,
			expected:   "https://semstreams.c360.io/robotics#Mission",
		},
		{
			name:       "non-existent type returns empty",
			entityType: message.EntityType{Domain: "robotics", Type: "unknown"},
			expected:   "",
		},
		{
			name:       "empty type returns empty",
			entityType: message.EntityType{},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEntityTypeIRI(tt.entityType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test that EntityTypeIRIs map is consistent with constants
func TestEntityTypeIRIsConsistency(t *testing.T) {
	// Verify all constants are present in the map
	constantTypes := []message.EntityType{
		EntityTypeDrone,
		EntityTypeMission,
		EntityTypeOperator,
		EntityTypeFormation,
		EntityTypeBattery,
		EntityTypePosition,
		EntityTypeAttitude,
		EntityTypeCriticalStatus,
	}

	for _, entityType := range constantTypes {
		_, exists := EntityTypeIRIs[entityType.Key()]
		assert.True(t, exists, "EntityTypeIRIs map should contain entry for constant %s", entityType.Key())
	}

	// Verify map doesn't contain unexpected entries
	assert.Equal(t, len(constantTypes), len(EntityTypeIRIs), 
		"EntityTypeIRIs map should have same number of entries as constants")
}

// Benchmark the GetEntityTypeIRI function for performance verification
func BenchmarkGetEntityTypeIRI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetEntityTypeIRI(EntityTypeDrone)
	}
}