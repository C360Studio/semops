# Robotics Processor README

**Last Updated**: 2024-08-27  
**Maintainer**: SemStreams Robotics Team

## Purpose & Scope

**What this component does**: Parses MAVLink protocol messages from autonomous vehicles (drones, ground vehicles, boats, submarines) and converts them into semantic payloads for downstream graph processing.

**Key responsibilities**:
- Parse binary MAVLink v1/v2 protocol messages from vehicle telemetry streams
- Convert JSON-formatted robotics data into structured payloads
- Create semantic payloads (heartbeat, position, battery, attitude) with Graphable interface
- Publish BaseMessage-wrapped payloads to NATS subjects for GraphProcessor consumption
- Filter messages by system ID, component ID, and message types
- Validate MAVLink checksums and handle malformed data gracefully
- Self-register payload types for automatic unmarshaling

**NOT responsible for**: Raw UDP packet handling, WebSocket connections, graph entity storage, rule processing, or UI visualization.

### Integration Points
- **Consumes from**: `raw.*.mavlink`, `raw.*.json` NATS subjects (from UDP input components)
- **Provides to**: GraphProcessor via `semantic.robotics.*` NATS subjects
- **External dependencies**: NATS client, MAVLinkParser, PayloadRegistry for payload self-registration

### Data Flow
```
UDP Packets → UDPInput → raw.drone.mavlink → RoboticsProcessor → semantic.robotics.* → GraphProcessor
                      → raw.drone.json   → RoboticsProcessor → semantic.robotics.* → RuleProcessor
```

### Configuration
```json
{
  "input_subjects": ["raw.*.mavlink", "raw.*.json"],
  "output_prefix": "robotics",
  "system_id_filter": [],
  "component_id_filter": [],
  "process_heartbeat": true,
  "process_position": true,
  "process_battery": true,
  "process_attitude": true,
  "process_gps": true,
  "validate_checksum": true,
  "drop_invalid": true
}
```

## Critical Behaviors (Testing Focus)

### Happy Path - What Should Work

1. **MAVLink Binary Message Processing**: Parses MAVLink v1/v2 binary data into semantic payloads
   - **Input**: Binary MAVLink packet with valid checksum from `raw.*.mavlink` subject
   - **Expected**: Creates typed payload (HeartbeatPayload, PositionPayload, etc.) and publishes BaseMessage to `semantic.robotics.*`
   - **Verification**: BaseMessage published with correct schema, payload implements Graphable interface, NATS message received by subscribers

2. **JSON Message Processing**: Converts JSON telemetry data into semantic payloads
   - **Input**: JSON message with `system_id` and telemetry fields from `raw.*.json` subject
   - **Expected**: Identifies message type, creates appropriate payload, wraps in BaseMessage, publishes to semantic subject
   - **Verification**: Correct payload type created, all JSON fields mapped to payload properties, BaseMessage structure valid

3. **Payload Self-Registration**: Automatically registers payloads for unmarshaling
   - **Input**: Package import triggers init() functions in payload files
   - **Expected**: All payloads (HeartbeatPayload, PositionPayload, BatteryPayload, AttitudePayload) registered with PayloadRegistry
   - **Verification**: PayloadRegistry contains entries for domain=robotics, category=heartbeat/position/battery/attitude, factory functions return correct types

4. **Message Filtering**: Filters messages based on system/component IDs and message types
   - **Input**: MAVLink packets with various system IDs, disabled message types in config
   - **Expected**: Only messages matching filters are processed, others are dropped silently
   - **Verification**: Filtered messages not published, enabled messages processed normally, no errors for filtered messages

5. **Checksum Validation**: Validates MAVLink message integrity before processing
   - **Input**: MAVLink packet with correct/incorrect checksum
   - **Expected**: Valid checksums processed normally, invalid checksums dropped or logged based on config
   - **Verification**: Invalid checksum messages not published, checksum error counters incremented, valid messages processed

### Error Conditions - What Should Fail Gracefully

1. **Malformed MAVLink Data**: Invalid sync bytes, truncated packets, corrupted headers
   - **Trigger**: Send binary data with wrong sync byte (not 0xFE/0xFD), incomplete packet, invalid length
   - **Expected**: Parser skips to next sync byte, logs parsing error, does not crash or publish invalid data
   - **Recovery**: Continue processing subsequent valid packets, maintain parser state consistency

2. **Missing NATS Connection**: Processor started without valid NATS client
   - **Trigger**: Call NewRoboticsProcessor(nil) or Start() with disconnected NATS
   - **Expected**: Returns ErrNoConnection error, processor does not start, no goroutines leaked
   - **Recovery**: Processor can be recreated with valid connection, previous instance properly cleaned up

3. **JSON Parsing Failures**: Invalid JSON structure, missing required fields, type mismatches
   - **Trigger**: Send malformed JSON, JSON without system_id, incorrect field types
   - **Expected**: Returns ErrInvalidData, message dropped, parsing continues for other messages
   - **Recovery**: Subsequent valid JSON messages processed normally, error counters updated

4. **Payload Validation Failures**: Invalid timestamps, out-of-range values, missing required fields
   - **Trigger**: Create payload with zero timestamp, negative system_id, invalid MAVLink version
   - **Expected**: Payload.Validate() returns specific error, message not published, validation details logged
   - **Recovery**: Other payloads with valid data processed normally, validation errors isolated per message

### Edge Cases - Boundary Conditions
- **High Message Volume**: 1000+ messages/second, buffer management, goroutine pool limits
- **Concurrent Start/Stop**: Rapid component lifecycle changes, subscription cleanup, resource deallocation
- **Mixed MAVLink Versions**: v1 and v2 packets in same stream, version-specific parsing, checksum differences
- **Unknown Message IDs**: MAVLink messages not in parser specification, graceful unknown message handling
- **Network Partitions**: NATS reconnection scenarios, message loss detection, state recovery

## Usage Patterns

### Typical Usage (How Other Code Uses This)

```go
// Create robotics processor with NATS dependency
config := component.ComponentConfig{
    Parameters: map[string]any{
        "input_subjects": []string{"raw.*.mavlink", "raw.*.json"},
        "output_prefix": "robotics",
        "process_heartbeat": true,
        "process_position": true,
        "validate_checksum": true,
    },
    NATSClient: natsClient,
}

processor, err := CreateRoboticsProcessor(ctx, config)
if err != nil {
    return fmt.Errorf("create robotics processor: %w", err)
}

// Initialize and start
if err := processor.Initialize(); err != nil {
    return fmt.Errorf("initialize processor: %w", err)
}

if err := processor.Start(ctx); err != nil {
    return fmt.Errorf("start processor: %w", err)
}

// Processor now subscribes to NATS subjects and processes messages
```

### Common Integration Patterns
- **Pipeline Pattern**: UDP → Robotics → Graph → Rules → WebSocket (sequential processing)
- **Fan-out Pattern**: Robotics publishes to multiple consumers (GraphProcessor, RuleProcessor, LogProcessor)
- **Registration Pattern**: Payloads self-register via init() functions for automatic discovery

## Testing Strategy

### Test Categories
1. **Unit Tests**: Individual payload creation, MAVLink parsing, message routing logic
2. **Integration Tests**: End-to-end with real NATS, actual MAVLink packets, payload registry
3. **Performance Tests**: High-throughput message processing, memory usage, goroutine management

### Test Quality Standards
- ✅ Tests MUST use real MAVLink binary data (not just synthetic structs)
- ✅ Tests MUST verify BaseMessage publication to NATS (not just payload creation)
- ✅ Tests MUST validate Graphable interface implementation on all payloads
- ✅ Tests MUST check payload self-registration in PayloadRegistry
- ✅ Tests MUST verify thread safety under concurrent message processing
- ❌ NO signature-only tests that just check function exists
- ❌ NO tests that mock away the MAVLink parsing (defeats the purpose)
- ❌ NO tests that ignore BaseMessage wrapper (critical for downstream consumers)

### Mock vs Real Dependencies
- **Use real dependencies for**: NATS client, MAVLink parser, payload validation, JSON unmarshaling
- **Use mocks for**: Network UDP sockets, file system operations, external HTTP APIs
- **Use testcontainers for**: NATS server for integration tests

## Metrics

This component exposes the following Prometheus metrics to monitor MAVLink parsing performance and identify processing bottlenecks:

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| semstreams_robotics_messages_received_total | Counter | Total raw messages received from NATS | subject, format |
| semstreams_robotics_messages_processed_total | Counter | Messages successfully parsed and published | system_id, message_type, format |
| semstreams_robotics_messages_dropped_total | Counter | Messages dropped due to parsing errors | system_id, error_type, format |
| semstreams_robotics_parse_duration_seconds | Histogram | Time to parse individual messages | format, message_type |
| semstreams_robotics_publish_duration_seconds | Histogram | Time to publish semantic messages to NATS | subject, payload_type |
| semstreams_robotics_checksum_errors_total | Counter | MAVLink messages with invalid checksums | system_id, message_id |
| semstreams_robotics_unknown_messages_total | Counter | MAVLink messages with unknown message IDs | system_id, message_id |
| semstreams_robotics_payload_size_bytes | Histogram | Size distribution of parsed payloads | payload_type |
| semstreams_robotics_active_vehicles | Gauge | Number of unique vehicles recently seen | - |
| semstreams_robotics_last_message_timestamp | Gauge | Unix timestamp of last processed message | system_id, message_type |

### Key Performance Indicators
- **Throughput**: `rate(semstreams_robotics_messages_processed_total[1m])` - Messages/sec being processed
- **Parse Success Rate**: `rate(semstreams_robotics_messages_processed_total[1m]) / rate(semstreams_robotics_messages_received_total[1m])` - Percentage of messages successfully parsed
- **Parse Latency**: `histogram_quantile(0.95, semstreams_robotics_parse_duration_seconds)` - P95 parsing latency
- **Checksum Error Rate**: `rate(semstreams_robotics_checksum_errors_total[1m])` - Invalid messages/sec
- **Vehicle Activity**: `semstreams_robotics_active_vehicles` - Number of vehicles actively sending telemetry

### Example Prometheus Queries
```promql
# Current processing throughput by message type
sum(rate(semstreams_robotics_messages_processed_total[1m])) by (message_type)

# Parse success rate across all vehicles
sum(rate(semstreams_robotics_messages_processed_total[1m])) / sum(rate(semstreams_robotics_messages_received_total[1m])) * 100

# Vehicles with high checksum error rates
sum(rate(semstreams_robotics_checksum_errors_total[5m])) by (system_id) > 0.1

# Parse latency by format (binary vs JSON)
histogram_quantile(0.99, sum(semstreams_robotics_parse_duration_seconds) by (format, le))

# Detect inactive vehicles
time() - semstreams_robotics_last_message_timestamp > 60
```

### Alerting Rules
```yaml
# High message drop rate
- alert: RoboticsProcessorHighDropRate
  expr: rate(semstreams_robotics_messages_dropped_total[5m]) / rate(semstreams_robotics_messages_received_total[5m]) > 0.05
  for: 2m
  annotations:
    summary: "Robotics processor dropping >5% of messages"

# Checksum errors indicating data corruption
- alert: RoboticsProcessorChecksumErrors
  expr: rate(semstreams_robotics_checksum_errors_total[5m]) > 1
  for: 1m
  annotations:
    summary: "High MAVLink checksum errors for system {{ $labels.system_id }}"

# Vehicle went offline
- alert: VehicleInactive
  expr: time() - semstreams_robotics_last_message_timestamp > 300
  for: 2m
  annotations:
    summary: "Vehicle {{ $labels.system_id }} inactive for 5+ minutes"

# Parse latency too high
- alert: RoboticsProcessorSlowParsing
  expr: histogram_quantile(0.95, semstreams_robotics_parse_duration_seconds) > 0.01
  for: 3m
  annotations:
    summary: "Robotics parser P95 latency >10ms"
```

## Implementation Notes

### Thread Safety
- **Concurrency model**: Multi-goroutine message processing with shared processor state
- **Shared state**: Config, statistics, subscriptions protected by RWMutex
- **Critical sections**: Message processing increments counters, subscription management requires exclusive locks

### Performance Considerations
- **Expected throughput**: 100-1000 messages/second typical, 10,000/second peak
- **Memory usage**: ~50MB baseline, ~1KB per buffered message, bounded buffer sizes
- **Bottlenecks**: MAVLink checksum calculation, JSON marshaling, NATS publishing latency

### Error Handling Philosophy
- **Error propagation**: Parsing errors logged but don't stop processing, fatal errors (no NATS) prevent startup
- **Retry strategy**: No retries for malformed data, NATS client handles connection retries
- **Circuit breaking**: Invalid data drops silently (configurable), persistent errors disable processor

## Troubleshooting

### Common Issues

1. **No messages published to semantic subjects**: Parser not receiving raw messages
   - **Cause**: UDP input not configured, NATS subject mismatch, processor not subscribed
   - **Solution**: Check NATS subscription patterns, verify UDP input publishing to raw.*.mavlink

2. **Checksum validation failures**: High invalid packet rate, no valid messages
   - **Cause**: MAVLink data corruption, wrong message specifications, CRC calculation errors
   - **Solution**: Verify MAVLink source integrity, check parser message specs match sender

3. **Memory usage growing over time**: Buffer accumulation, goroutine leaks
   - **Cause**: Processing slower than ingestion, subscription cleanup failures, parsing buffer growth
   - **Solution**: Monitor processor.DataFlow() metrics, check goroutine count, restart processor

4. **Unknown message types**: Many unrecognized MAVLink message IDs
   - **Cause**: Custom MAVLink messages, newer protocol versions, incomplete message specifications
   - **Solution**: Register custom message specs, update parser to latest MAVLink standard

### Debug Information
- **Logs to check**: `[RoboticsProcessor]` prefix messages, checksum validation errors, parsing statistics
- **Metrics to monitor**: `messages_processed`, `errors`, `checksum_errors`, `unknown_messages` counters
- **Health checks**: processor.Health().Healthy, NATS connection status, subscription count

## Development Workflow

### Before Making Changes
1. Understand MAVLink protocol and message structure for affected message types
2. Check payload interfaces (Graphable, Observable, Timestampable) for compatibility
3. Verify BaseMessage wrapper structure matches downstream consumer expectations
4. Review parser message specifications for accuracy

### After Making Changes
1. Run MAVLink integration tests with real binary data
2. Verify payload self-registration still works (check PayloadRegistry)
3. Test BaseMessage publishing and downstream consumption
4. Update message specifications if new MAVLink messages added
5. Performance test with high message volume

## Related Documentation
- [MAVLink Protocol Specification](https://mavlink.io/en/messages/)
- [SemStreams Message Architecture](/docs/architecture/MESSAGE_ARCHITECTURE.md)
- [GraphProcessor Integration](/pkg/processor/graph/README.md)
- [Payload Registration System](/pkg/component/README.md)

## Directory Structure

The robotics processor package follows a modular structure supporting MAVLink protocol processing:

```
pkg/processor/robotics/
├── README.md                    # This comprehensive documentation
├── processor.go                 # Main RoboticsProcessor implementation with LifecycleComponent
├── subjects.go                  # NATS subject constants for semantic routing
├── types.go                     # Domain-specific type definitions
├── constants/
│   └── mavlink.go              # MAVLink protocol constants and message IDs
├── parser/
│   ├── mavlink_parser.go       # Robust MAVLink v1/v2 binary parser
│   └── *_test.go              # Comprehensive parser unit tests
├── payloads/
│   ├── heartbeat.go            # HeartbeatPayload with system status + Graphable interface
│   ├── position.go             # PositionPayload with GPS coordinates + Graphable interface
│   ├── battery.go              # BatteryPayload with power status + Graphable interface  
│   ├── attitude.go             # AttitudePayload with orientation data + Graphable interface
│   └── *_test.go              # Payload behavioral interface tests
├── rules/
│   └── battery_monitor.go      # Battery monitoring business rules
├── vocabulary/
│   ├── entities.go             # Robotics entity types (Drone, Battery, etc.)
│   ├── relationships.go        # Relationship types (CONTROLLED_BY, MEMBER_OF, etc.)
│   └── confidence.go           # Confidence level constants
└── *_test.go                   # Integration and compliance tests
```

## Key Architecture Patterns

### 1. Robust Protocol Parsing
The MAVLinkParser implements stateful, buffer-based parsing with comprehensive error handling:

```go
type MAVLinkParser struct {
    buffer        []byte                 // Stateful buffer for partial packets
    checksumTable [256]uint16           // CRC-16-CCITT lookup table  
    messageSpecs  map[uint32]*MessageSpec // Message format specifications
    parsingStats  ParsingStats           // Performance and error metrics
}
```

**Key Features**: Handles partial packets, validates checksums, tracks unknown message types, maintains parsing statistics.

### 2. Semantic Payload System
All payloads implement behavioral interfaces for rich semantic processing:

```go
type HeartbeatPayload struct {
    SystemID     uint8     `json:"system_id"`
    Timestamp    time.Time `json:"timestamp"`
    VehicleType  uint8     `json:"vehicle_type"`
    // ... MAVLink fields
}

// Behavioral interfaces
func (h *HeartbeatPayload) EntityID() string { return fmt.Sprintf("drone_%d", h.SystemID) }
func (h *HeartbeatPayload) Entities() []message.EntityHint { /* Creates drone entity */ }
func (h *HeartbeatPayload) Relationships() []message.RelationshipHint { /* Control relationships */ }
```

**Interfaces Implemented**: Graphable (entity extraction), Observable (what's being observed), Timestampable (temporal context).

### 3. BaseMessage Wrapper Pattern
All semantic payloads are wrapped in BaseMessage for consistent downstream processing:

```go
// Parser creates semantic payload
heartbeat := payloads.NewHeartbeatPayload(systemID, componentID, time.Now())

// Wrap in BaseMessage with proper schema
msg := message.NewBaseMessage(
    heartbeat.Schema(), // MessageType{Domain:"robotics", Category:"heartbeat", Version:"v1"}
    heartbeat,          // The actual payload
    "robotics-processor", // Source identifier
)

// Publish complete BaseMessage (not just payload)
msgBytes, _ := json.Marshal(msg)
nc.Publish("semantic.robotics.heartbeat", msgBytes)
```

This ensures GraphProcessor can unmarshal BaseMessages using the PayloadRegistry.