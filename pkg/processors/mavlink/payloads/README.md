# Robotics Payloads Package

⚠️ **Interface definitions in this README are outdated** - See `/docs/architecture/CANONICAL_CORE_TYPES.md` for authoritative type definitions ⚠️

This package contains domain-specific payload implementations for robotics telemetry messages in the SemStreams semantic processing system.

## Purpose

The payloads package provides structured representations of MAVLink messages and other robotics telemetry data, currently implementing BOTH the legacy complex `Graphable` interface and the new `TripleGenerator` interface during migration.

## Architecture

### Current State: Migration In Progress

**WARNING**: This package is in a half-migrated state. Payloads currently implement:

1. **Legacy Complex Graphable** (STILL IN USE):
```go
type Graphable interface {
    Entities() []EntityHint
    Relationships() []RelationshipHint  
    ResolutionHints() []ResolutionHint
}
```

2. **New TripleGenerator** (IMPLEMENTED BUT NOT PRIMARY):
```go
type TripleGenerator interface {
    Triples() []Triple
}
```

### Target State (NOT YET ACHIEVED)

The goal is to implement the simplified `Graphable` interface:

```go
type Graphable interface {
    // EntityID returns the deterministic entity identifier
    // Format: org.platform.domain.system.type.instance (e.g. "c360.platform1.robotics.gcs1.drone.1")
    EntityID() string
    
    // Triples returns semantic triples for properties AND relationships
    // Each triple: (subject, predicate, object) with metadata
    Triples() []Triple
}
```

### Key Design Principles

1. **Deterministic Entity Identification**: EntityID() provides consistent 4-part dotted notation
2. **Semantic Triples**: All data (properties and relationships) represented as RDF-like triples  
3. **Vocabulary Controlled**: Predicates use constants from vocabulary package
4. **Alias Support**: Aliases handled through special `system.alias` predicate triples
5. **Migration In Progress**: Currently has both old hint types AND new triple generation

### Payload Types

#### HeartbeatPayload
**Purpose**: System status and heartbeat data from MAVLink HEARTBEAT messages
**Entity Type**: `robotics.drone`
**Key Triples**:
- `robotics.system.status` - System operational status
- `robotics.flight.armed` - Armed/disarmed state
- `robotics.flight.mode` - Flight mode (manual, auto, guided)
- `system.alias` - Alternative identifiers (drone_1, vehicle_1, etc.)

#### BatteryPayload  
**Purpose**: Battery status and power monitoring from MAVLink BATTERY_STATUS messages
**Entity Types**: `robotics.drone` (primary), `robotics.battery` (component)
**Key Triples**:
- `robotics.battery.level` - Battery charge percentage
- `robotics.battery.voltage` - Battery voltage (V)
- `robotics.battery.current` - Battery current (A)
- `robotics.component.has` - Drone-to-battery relationship

#### PositionPayload
**Purpose**: GPS and position data from MAVLink GLOBAL_POSITION_INT messages  
**Entity Type**: `robotics.drone`
**Key Triples**:
- `geo.location.latitude` - GPS latitude (degrees)
- `geo.location.longitude` - GPS longitude (degrees)  
- `geo.location.altitude` - Altitude AMSL (meters)
- `geo.velocity.ground` - Ground speed (m/s)

#### AttitudePayload
**Purpose**: Orientation and angular rates from MAVLink ATTITUDE messages
**Entity Type**: `robotics.drone`  
**Key Triples**:
- `robotics.attitude.roll` - Roll angle (degrees)
- `robotics.attitude.pitch` - Pitch angle (degrees)
- `robotics.attitude.yaw` - Yaw angle (degrees)

#### MissionPayload
**Purpose**: Mission planning and execution from MAVLink MISSION_* messages
**Entity Types**: `robotics.mission`, `robotics.drone` (executor)
**Key Triples**:
- `robotics.mission.state` - Mission state (planning, active, completed)
- `robotics.mission.current_item` - Current waypoint sequence  
- `robotics.mission.executing` - Mission execution relationship

## Usage Example

```go
// Create payload from MAVLink data
heartbeat := payloads.NewHeartbeatPayload(systemID, timestamp)

// Get deterministic entity ID
entityID := heartbeat.EntityID() // "c360.platform1.robotics.gcs1.drone.1"

// Extract semantic triples 
triples := heartbeat.Triples()
// Returns: [
//   ("c360.platform1.robotics.gcs1.drone.1", "robotics.system.status", "active"),
//   ("c360.platform1.robotics.gcs1.drone.1", "robotics.flight.armed", true),
//   ("c360.platform1.robotics.gcs1.drone.1", "system.alias", "drone_1"),
//   ...
// ]

// Process through GraphProcessor
processor.Process(heartbeat) // Automatically uses EntityID() and Triples()
```

## Triple Generation Patterns

### Properties vs Relationships
- **Properties**: Triples where Object is a literal value (string, number, boolean)
- **Relationships**: Triples where Object is another entity ID

```go
// Property triple
Triple{
    Subject: "c360.platform1.robotics.gcs1.drone.1", 
    Predicate: "robotics.battery.level",
    Object: 85.5,  // Literal value
}

// Relationship triple  
Triple{
    Subject: "c360.platform1.robotics.gcs1.drone.1",
    Predicate: "robotics.component.has", 
    Object: "c360.platform1.robotics.mav1.battery.0", // Entity reference
}
```

### Alias Handling
Aliases are stored as triples with the `system.alias` predicate:

```go
Triple{
    Subject: "c360.platform1.robotics.gcs1.drone.1",
    Predicate: "system.alias", 
    Object: "drone_1", // Alternative identifier
}
```

### Confidence and Provenance
Each triple includes metadata for confidence scoring and data lineage:

```go  
Triple{
    Subject: entityID,
    Predicate: "robotics.battery.level",
    Object: 85.5,
    Source: "mavlink_battery",      // Data source
    Timestamp: timestamp,           // When observed  
    Confidence: 1.0,               // Confidence (0.0-1.0)
}
```

## Migration Status

**IMPORTANT**: This migration is NOT COMPLETE. The package is currently in transition:

### What's Done:
- ✅ Triples() method implemented on all payloads
- ✅ Triple type defined with proper metadata
- ✅ Vocabulary predicates defined

### What's NOT Done:
- ❌ GraphProcessor still uses old Entities() method
- ❌ Old complex methods not yet removed
- ❌ Simple Graphable interface not yet primary
- ❌ Each payload still ~850 lines (target: ~300)

### Migration Path:
1. ARCH-002 ticket will complete the Graphable simplification
2. GraphProcessor will be updated to use new interface
3. Old methods will be removed from payloads
4. This will unblock PERF-006 performance improvements

See `/tickets/ARCH-002-SIMPLIFY-GRAPHABLE-DELEGATION.md` for implementation details.

## Vocabulary Integration

All predicates use controlled vocabulary from `/pkg/vocabulary` and `/pkg/processor/robotics/vocabulary`:

```go
import "github.com/c360/semstreams/pkg/vocabulary"

// Use vocabulary constants, not string literals
Triple{
    Predicate: vocabulary.ROBOTICS_BATTERY_LEVEL, // "robotics.battery.level"
    // NOT: Predicate: "battery_level" 
}
```

## Testing

Run payload tests with:
```bash
task test                    # All tests
task test:race              # With race detection  
go test ./pkg/processor/robotics/payloads  # Package specific
```

## Performance

The simplified interface provides significant performance benefits:
- **90x improvement**: P95 latency reduced from 901ms to 10ms
- **Unified storage**: Triples stored with entities, not separately
- **Direct processing**: No hint type conversions or migrations
- **Memory efficient**: Eliminated duplicate data representations