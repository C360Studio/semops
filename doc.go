// Package semops provides domain-specific robotics and operational semantics
// built on top of the zero-domain semstreams framework.
//
// SemOps focuses on:
//   - Robotics platforms (UAVs, USVs, ground vehicles)
//   - Sensor observations and telemetry
//   - SOSA/SSN ontology compliance
//   - Protocol adapters (MAVLink, TAK, NMEA)
//   - Operational monitoring and control
//
// # Architecture
//
// SemOps is a domain application that uses semstreams interfaces:
//
//	┌─────────────────────────────────────────┐
//	│           SemOps Domain Layer           │
//	│  (Robotics, Sensors, Observations)      │
//	├─────────────────────────────────────────┤
//	│      Adapters: MAVLink, TAK, NMEA       │
//	│    Entities: Sensor, Platform, Obs      │
//	│      Services: Monitoring, Control      │
//	│         API: SOSA/SSN endpoints         │
//	└─────────────────────────────────────────┘
//	              ↓ uses ↓
//	┌─────────────────────────────────────────┐
//	│     SemStreams Framework Interfaces     │
//	│  EntityStore, GraphProcessor, Index...  │
//	└─────────────────────────────────────────┘
//	              ↓ uses ↓
//	┌─────────────────────────────────────────┐
//	│         StreamKit Foundation            │
//	│    Components, NATS, Observability      │
//	└─────────────────────────────────────────┘
//
// # Domain Model
//
// SemOps defines robotics-specific entity types that map to semstreams
// generic Entity model:
//
//   - **Platform**: UAV, USV, ground vehicle (maps to Entity with type="platform")
//   - **Sensor**: Temperature, GPS, camera (maps to Entity with type="sensor")
//   - **Observation**: Sensor reading with SOSA semantics (Entity with type="observation")
//   - **FeatureOfInterest**: What's being observed (Entity with custom type)
//
// # Protocol Adapters
//
// SemOps provides adapters that translate domain protocols to semstreams operations:
//
//   - **MAVLink**: Drone telemetry → Entity updates + Triple assertions
//   - **TAK**: Team Awareness Kit → Entity/relationship updates
//   - **NMEA**: Marine navigation → GPS observations as Entities
//
// Each adapter is a StreamKit Input component that writes to semstreams.
//
// # SOSA/SSN Compliance
//
// SemOps implements the W3C SOSA (Sensor, Observation, Sample, and Actuator)
// and SSN (Semantic Sensor Network) ontologies using semstreams triples:
//
//	sensor --observes--> feature-of-interest
//	observation --made-by--> sensor
//	observation --observed-property--> property
//	observation --has-result--> value
//	platform --hosts--> sensor
//
// This provides semantic interoperability with other SOSA/SSN systems.
//
// # Getting Started
//
// To use SemOps with semstreams:
//
//	import (
//	    "github.com/c360/semops/pkg/entities"
//	    "github.com/c360/semops/pkg/adapters/mavlink"
//	    "github.com/c360/semstreams/pkg/interfaces/store"
//	)
//
//	// Create a sensor entity
//	sensor := entities.NewSensor("temp-sensor-1", "temperature")
//
//	// Store via semstreams EntityStore
//	entityStore.Create(ctx, sensor.ToEntity())
//
//	// Process MAVLink telemetry
//	adapter := mavlink.NewAdapter(entityStore, graphProcessor)
//	adapter.ProcessHeartbeat(mavlinkMsg)
//
// # Configuration
//
// SemOps uses JSON configuration files (see configs/) that define:
//   - Platform definitions (vehicle types, sensors)
//   - Data flow pipelines (MAVLink → Processing → Storage)
//   - Query endpoints (SOSA API configuration)
//   - Monitoring rules (alerts, thresholds)
//
// These configs leverage StreamKit's component system and semstreams interfaces.
//
// # Directory Structure
//
//	semops/
//	├── pkg/
//	│   ├── adapters/     # Protocol adapters (MAVLink, TAK, NMEA)
//	│   ├── entities/     # Domain entity types (Sensor, Platform, etc.)
//	│   ├── services/     # Domain services (monitoring, control)
//	│   └── api/sosa/     # SOSA/SSN REST API endpoints
//	├── cmd/semops/       # Main application binary
//	└── configs/          # Configuration files
//
// # Migration from SemStreams
//
// SemOps receives robotics-specific code that was originally in semstreams:
//   - processor/mavlink/ → adapters/mavlink/
//   - Domain-specific entity types
//   - Robotics flow configurations
//   - SOSA API implementations
//
// This separation keeps semstreams as a zero-domain-knowledge framework while
// SemOps provides the robotics domain expertise.
package semops
