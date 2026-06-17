// Package semops is the SemStreams-backed data-fusion common operating picture
// product.
//
// SemOps owns the product layer around operational feeds, COP entities, scenario
// playback, operator APIs, and the Svelte COP UI. SemStreams owns the framework
// substrate: graph mutation/query contracts, projection contracts, ownership
// registration, indexing profiles, rule execution, and NATS/JetStream runtime
// primitives.
//
// # Current Revival Direction
//
// The repository is being revived as a greenfield COP product. Old robotics and
// MAVLink code is salvage, not the default architecture. Legacy MAVLink/SITL
// references have been deleted after useful parser, generator, command, and raw
// lane material moved behind modern SemOps package boundaries.
//
// The first product model focuses on:
//
//   - tracks
//   - assets
//   - hazard areas
//   - sensor footprints
//   - alerts
//   - tasks
//   - advisories
//
// Feed adapters should write through SemStreams projection and ownership
// contracts. High-rate native packets should remain on bounded raw lanes, while
// the graph receives current state, durable events, provenance, confidence, and
// relationship evidence.
//
// # Feed Boundaries
//
// The first structural feeds are MAVLink and TAK/CoT, followed by CAP/EDXL once
// the COP entity model and validation gates are stable. ADS-B, SAPIENT, and
// KLV/STANAG 4609 remain evidence-gated until fixtures, replay, compliance, and
// binary-handling proof points are available.
//
// # Documentation
//
// The OpenSpec change for this revival lives under:
//
//	openspec/changes/revive-cop-product
//
// The architecture baseline is:
//
//	docs/cop-demo-revival-architecture.md
//
// Legacy cleanup details are:
//
//	docs/legacy-quarantine.md
package semops
