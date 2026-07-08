## 1. Contract Shape

- [x] 1.1 Draft `docs/contracts/semlink-companion-readback-v0.md` with native request, response, error, duplicate, and
      status sections
- [x] 1.2 Add a standards mapping table for every native contract field
- [x] 1.3 Mark every C360-only field as a governance exception with a reason and owning authority
- [x] 1.4 Add contract fixtures for accepted, duplicate, rejected, and expired/stale readback requests

## 2. SemOps Admission Proof

- [x] 2.1 Add focused tests that load the native accepted fixture and project it through existing SemLink readback
      ingress into governed command intent
- [x] 2.2 Add negative tests for missing authority scope, companion-node mismatch, unsupported MAVLink command/message,
      unborn target, duplicate idempotency key, and stale TTL
- [x] 2.3 Prove the response preserves no-native/no-companion-transmit posture and graph admission evidence

## 3. SemConnect Interop Proof

- [x] 3.1 Draft CS API projection fixtures for the accepted SemLink readback intent using System, ControlStream,
      Command, and status/event/readback resources where they fit
- [x] 3.2 Verify the CS API projection can carry MAVLink command/message IDs, target system/component IDs,
      correlation, sender/provenance, status, and result/readback evidence without semantic loss
- [x] 3.3 Record any SemConnect capability gaps as follow-up asks rather than adding SemOps-local CS API behavior

## 4. SemLink Hold-Out Review

- [x] 4.1 Publish the draft native companion contract and fixtures for SemLink feedback before contract acceptance
- [x] 4.2 Ask SemLink to validate whether the native companion contract can be implemented from its companion
      evidence/runtime state without CS API in the hot path
- [x] 4.3 Ask SemLink to identify any fields that add companion-runtime friction or duplicate existing MAVLink/BlueOS
      concepts
- [x] 4.4 Classify SemLink feedback as blocking, friction, interop, or deferred
- [x] 4.5 Update the contract, mark a non-goal, or create follow-up tasks for each blocking or friction item before
      SemLink resumes integration implementation
- [x] 4.6 Record a SemLink feedback/disposition review under the spike reviews folder

## 5. Friction Decision

- [x] 5.1 Score CS API/SemConnect as Green, Amber, or Red for the SemLink-to-SemOps path
- [x] 5.2 Record the MVP decision only after SemLink feedback has been dispositioned: native hot path, optional CS API
      path, or CS API-first path
- [x] 5.3 Run `openspec validate semlink-semops-contract-spike --strict`
