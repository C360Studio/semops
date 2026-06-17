// Package mavlink hosts the modern SemOps MAVLink adapter boundary.
//
// The active slice contains a MAVLink v1/v2 parser, a MAVLink v2 scenario
// generator, bounded raw-frame lane, replay fixture store, command frame
// helpers, and real-frame tests for heartbeat, global position, attitude,
// battery status, COMMAND_LONG, and COMMAND_ACK. The old ignored SITL
// controller/scenario files were deleted after useful command encoding and ACK
// parsing moved here.
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
// The next simulator step should add a modern SITL or MAVSDK harness with
// explicit readiness and state polling before SemOps claims live command/control
// coverage.
package mavlink
