// Package mavlink hosts the modern SemOps MAVLink adapter boundary.
//
// The first active slice contains a MAVLink v1/v2 parser, a MAVLink v2
// scenario generator, and real-frame tests for heartbeat, global position,
// attitude, and battery status. Legacy SITL files remain under
// pkg/processors/mavlink with the ignore build constraint until their useful
// command/control behavior is extracted here or deleted.
//
// # Adapter Contract
//
// The adapter must:
//
//   - decode real MAVLink frames before projection
//   - keep raw high-rate frames on a bounded lane
//   - birth source asset and track entities explicitly with
//     graph.CreateEntityWithTriplesRequest
//   - project current vehicle state with graph.UpdateEntityWithTriplesRequest
//   - use pkg/cop projection contracts for ownership, indexing profiles, and
//     ADR-056 foreign-edge declarations
//   - never rely on triple.add or triple.add_batch auto-vivify
//
// MAVLink and TAK track-source edges are born-first EdgeStrict relationships:
// the target source asset must exist before cop.track.source is written.
//
// # First Coverage
//
// The next extraction should preserve SITL command evidence only where tests
// prove the behavior from real frames and explicit simulator gates.
package mavlink
