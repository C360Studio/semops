# SemOps - Robotics Operations on SemStreams

**SemOps** is a domain-specific application built on the [SemStreams](../semstreams/) semantic framework, providing robotics and operational semantics for autonomous systems.

## Overview

SemOps separates robotics domain knowledge from the zero-domain SemStreams framework:

```
┌─────────────────────────────────────────┐
│           SemOps (This Module)          │
│     Robotics, Sensors, SOSA/SSN        │
├─────────────────────────────────────────┤
│  • UAV/USV/UGV platforms               │
│  • Sensor observations                  │
│  • MAVLink/TAK/NMEA adapters           │
│  • SOSA API endpoints                   │
└─────────────────────────────────────────┘
              ↓ uses ↓
┌─────────────────────────────────────────┐
│          SemStreams Framework           │
│  Zero-domain semantic knowledge graph   │
├─────────────────────────────────────────┤
│  • EntityStore interface                │
│  • GraphProcessor interface             │
│  • IndexEngine interface                │
│  • QueryEngine interface                │
└─────────────────────────────────────────┘
              ↓ uses ↓
┌─────────────────────────────────────────┐
│         StreamKit Foundation            │
│    Components, NATS, Observability      │
└─────────────────────────────────────────┘
```

## Architecture

### Domain Entities

SemOps defines robotics-specific entity types that map to SemStreams generic `Entity`:

- **Platform**: UAV, USV, UGV, ground station
- **Sensor**: Temperature, GPS, IMU, camera, lidar
- **Observation**: SOSA-compliant sensor readings
- **Mission**: Waypoint sequences, tasks
- **FeatureOfInterest**: What's being observed

### Protocol Adapters

Adapters translate domain protocols into SemStreams operations:

#### MAVLink Adapter
- **Input**: UDP MAVLink messages from drones
- **Output**: Entity updates + Triple assertions
- **Status**: 🚧 To be migrated from `semstreams/processor/mavlink/`

#### TAK Adapter (Future)
- **Input**: TAK/CoT messages
- **Output**: Entity updates for team awareness

#### NMEA Adapter (Future)
- **Input**: NMEA 0183 marine data
- **Output**: GPS observations as entities

### SOSA/SSN Compliance

SemOps implements [W3C SOSA](https://www.w3.org/TR/vocab-ssn/) (Sensor, Observation, Sample, and Actuator) ontology:

```
Platform --hosts--> Sensor
Sensor --observes--> FeatureOfInterest
Observation --made-by--> Sensor
Observation --observed-property--> Property
Observation --has-result--> Value
```

These relationships are stored as SemStreams triples.

## Directory Structure

```
semops/
├── go.mod                  # Module dependencies
├── doc.go                  # Package documentation
├── README.md               # This file
│
├── cmd/
│   └── semops/
│       └── main.go         # Main application entry point
│
├── pkg/
│   ├── entities/           # Domain entity types
│   │   ├── sensor.go       # Sensor entity (temperature, GPS, etc.)
│   │   ├── platform.go     # Platform entity (UAV, USV, etc.)
│   │   └── observation.go  # SOSA Observation entity
│   │
│   ├── adapters/           # Protocol adapters
│   │   ├── mavlink/        # MAVLink protocol (drones)
│   │   ├── tak/            # TAK/CoT protocol (future)
│   │   └── nmea/           # NMEA marine data (future)
│   │
│   ├── services/           # Domain services
│   │   └── monitoring.go   # Health monitoring, alerts
│   │
│   └── api/
│       └── sosa/           # SOSA/SSN REST API
│           └── handlers.go # Query endpoints
│
└── configs/
    └── robotics-flow.json  # StreamKit flow configuration
```

## Migration Plan

SemOps is receiving robotics-specific code from SemStreams:

### From SemStreams → To SemOps

| Source | Destination | Status |
|--------|-------------|--------|
| `semstreams/processor/mavlink/` | `semops/pkg/adapters/mavlink/` | 🚧 Pending |
| Domain entity types | `semops/pkg/entities/` | ✅ Scaffolded |
| SOSA API handlers | `semops/pkg/api/sosa/` | 🚧 Pending |
| Robotics flow configs | `semops/configs/` | 🚧 Pending |

### Refactoring Strategy

1. **Keep SemStreams Generic**: Remove robotics references from SemStreams
2. **Use Interfaces**: SemOps uses `EntityStore`, `GraphProcessor`, etc.
3. **StreamKit Components**: Adapters follow StreamKit patterns
4. **No Direct NATS**: All operations via SemStreams interfaces

## Getting Started

### Prerequisites

```bash
# SemStreams framework (from parent directory)
cd ../semstreams && go mod download

# StreamKit foundation
cd ../streamkit && go mod download
```

### Build

```bash
# From semops directory
go mod download
go build ./cmd/semops
```

### Run

```bash
./semops --config configs/robotics-flow.json
```

## Usage Example

```go
package main

import (
    "context"

    "github.com/c360/semops/pkg/entities"
    "github.com/c360/semstreams/pkg/impl/nats_store"
)

func main() {
    ctx := context.Background()

    // Create semstreams EntityStore
    entityStore, _ := nats_store.NewNATSEntityStore(ctx, handler, bucket)

    // Create a sensor entity
    sensor := entities.NewSensor(
        "c360.uav-alpha.ops.gcs1.sensor.temp-1",
        entities.SensorTypeTemperature,
    )
    sensor.Properties.Make = "Bosch"
    sensor.Properties.Model = "BME280"

    // Store via semstreams interface
    entityStore.Create(ctx, sensor.ToEntity())

    // Create an observation
    obs := entities.NewObservation(
        "c360.uav-alpha.ops.gcs1.observation.temp-12345",
        sensor.ID,
        "temperature",
        22.5, // celsius
    )
    obs.Result.Units = "celsius"

    // Store observation
    entityStore.Create(ctx, obs.ToEntity())
}
```

## Development

### Running Tests

```bash
go test ./...
```

### Code Quality

```bash
# Format code
gofmt -w .

# Lint
golangci-lint run

# Vet
go vet ./...
```

## Documentation

- **SemStreams Interfaces**: See `../semstreams/pkg/interfaces/`
- **StreamKit Components**: See `../streamkit/component/`
- **SOSA/SSN Ontology**: https://www.w3.org/TR/vocab-ssn/
- **MAVLink Protocol**: https://mavlink.io/

## License

Copyright (c) 2025 C360
