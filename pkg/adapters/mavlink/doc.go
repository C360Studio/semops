// Package mavlink provides a MAVLink protocol adapter for semstreams.
//
// This adapter translates MAVLink messages into semstreams Entity and Triple operations,
// enabling robotics platforms to participate in the semantic knowledge graph.
//
// # Architecture
//
// The MAVLink adapter is a StreamKit Input component that:
//  1. Receives MAVLink messages (UDP, serial, TCP)
//  2. Translates messages to Entity updates and Triple assertions
//  3. Writes to semstreams via EntityStore and GraphProcessor interfaces
//  4. Publishes events to NATS for downstream processing
//
// # Message Mapping
//
// MAVLink messages map to semstreams operations:
//
//	HEARTBEAT → Platform entity update (status, mode)
//	GPS_RAW_INT → Observation entity (GPS reading)
//	ATTITUDE → Observation entity (IMU data)
//	BATTERY_STATUS → Observation + Triple (battery → powers → platform)
//	MISSION_ITEM → Entity + Triple (waypoint → part-of → mission)
//
// # SOSA Compliance
//
// The adapter creates SOSA-compliant observations:
//
//	observation := entities.NewObservation(
//	    obsID,
//	    sensorID,        // GPS sensor
//	    "gps_position",  // observed property
//	    positionValue,   // lat/lon result
//	)
//
//	// Store observation entity
//	entityStore.Create(ctx, observation.ToEntity())
//
//	// Assert SOSA relationships
//	graphProcessor.AddTriple(ctx, &graph.Triple{
//	    Subject:   obsID,
//	    Predicate: "made-by",
//	    Object:    sensorID,
//	})
//
// # Migration from SemStreams
//
// This code will be migrated from:
//   semstreams/processor/mavlink/ → semops/pkg/adapters/mavlink/
//
// The refactored adapter will:
//   - Use semstreams interfaces (not direct NATS access)
//   - Follow StreamKit component patterns
//   - Maintain MAVLink protocol compatibility
//   - Support all existing message types
//
// # Configuration
//
// The adapter is configured via StreamKit component config:
//
//	{
//	  "component": "mavlink-adapter",
//	  "type": "input",
//	  "config": {
//	    "listen_address": "0.0.0.0:14550",
//	    "protocol": "udp",
//	    "system_id": 1,
//	    "component_id": 1
//	  }
//	}
package mavlink

// TODO: Migrate from semstreams/processor/mavlink/
// TODO: Implement StreamKit Input component interface
// TODO: Use semstreams EntityStore and GraphProcessor
// TODO: Add comprehensive tests with mock MAVLink messages
