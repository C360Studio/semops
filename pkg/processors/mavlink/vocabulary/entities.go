//go:build ignore
// +build ignore

package vocabulary

import "github.com/c360/semstreams/message"

// Entity types for the robotics domain using structured EntityType.
// These constants provide a controlled vocabulary for entity types to ensure
// consistency across robotics payloads and prevent typos in type definitions.
var (
	// Primary entities - main actors in the robotics domain
	EntityTypeDrone     = message.EntityType{Domain: "robotics", Type: "drone"}     // Autonomous or remote-controlled drone
	EntityTypeMission   = message.EntityType{Domain: "robotics", Type: "mission"}   // Mission or task being executed
	EntityTypeOperator  = message.EntityType{Domain: "robotics", Type: "operator"}  // Human operator or controller
	EntityTypeFormation = message.EntityType{Domain: "robotics", Type: "formation"} // Group formation or swarm coordination

	// Component entities - subsystems and components
	EntityTypeBattery  = message.EntityType{Domain: "robotics", Type: "battery"}  // Power system and battery status
	EntityTypePosition = message.EntityType{Domain: "robotics", Type: "position"} // GPS and positioning information
	EntityTypeAttitude = message.EntityType{Domain: "robotics", Type: "attitude"} // Orientation, pitch, roll, yaw

	// Status entities - operational and system states
	EntityTypeCriticalStatus = message.EntityType{Domain: "robotics", Type: "critical_status"} // Critical system status alerts
)

// EntityTypeIRIs maps entity types to their corresponding IRI representations.
// This provides IRI mappings for federation support without changing existing business logic.
var EntityTypeIRIs = map[string]string{
	EntityTypeDrone.Key():          "https://semstreams.c360.io/robotics#Drone",
	EntityTypeMission.Key():        "https://semstreams.c360.io/robotics#Mission",
	EntityTypeOperator.Key():       "https://semstreams.c360.io/robotics#Operator",
	EntityTypeFormation.Key():      "https://semstreams.c360.io/robotics#Formation",
	EntityTypeBattery.Key():        "https://semstreams.c360.io/robotics#Battery",
	EntityTypePosition.Key():       "https://semstreams.c360.io/robotics#Position",
	EntityTypeAttitude.Key():       "https://semstreams.c360.io/robotics#Attitude",
	EntityTypeCriticalStatus.Key(): "https://semstreams.c360.io/robotics#CriticalStatus",
}

// GetEntityTypeIRI returns the IRI for a given robotics entity type.
// Returns empty string if the entity type is not found or is invalid.
//
// This provides a convenient lookup for robotics-specific entity type IRIs
// for future federation support without impacting current system operations.
func GetEntityTypeIRI(entityType message.EntityType) string {
	if iri, exists := EntityTypeIRIs[entityType.Key()]; exists {
		return iri
	}
	return ""
}

// GetEntityTypeIRIByKey returns the IRI for a given entity type key (backwards compatibility)
func GetEntityTypeIRIByKey(key string) string {
	if iri, exists := EntityTypeIRIs[key]; exists {
		return iri
	}
	return ""
}
