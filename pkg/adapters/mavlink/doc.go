// Package mavlink will host the modern SemOps MAVLink adapter.
//
// The current package is intentionally a placeholder. Legacy MAVLink reference
// files remain under pkg/processors/mavlink with the ignore build constraint
// until their useful parser, generator, and SITL behavior is extracted here or
// deleted.
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
// The first extraction should preserve heartbeat, global position, attitude,
// battery, and SITL command evidence only where tests prove the behavior from
// real frames.
package mavlink
