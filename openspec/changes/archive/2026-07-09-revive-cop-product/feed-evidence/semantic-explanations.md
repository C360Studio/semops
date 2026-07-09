# Semantic Explanation Evidence

Status: deterministic core accepted for the COP revival change.

## Scope

`internal/semantic` derives read-only semantic explanation artifacts from a COP snapshot. The first builder emits:

- track translations for source-owned tracks;
- association anomaly explanations for review-worthy fusion association evidence;
- task prompt/input references, output text, source evidence, and semantic trajectory references on every item.

This is a product evidence object, not a hosted semantic service yet.

## Boundary

- Source feeds keep ownership of source tracks.
- Fusion keeps ownership of association evidence.
- Semantic explanations do not write graph state, mutate source entities, merge identities, transmit commands, or
  grant operator authority.
- The trajectory reference is a deterministic semantic derivation reference. The task input reference points back to
  the COP snapshot entity used as input.
- Semantic explanation output should remain `content` shaped if it is later projected into SemStreams.

## Accepted Evidence

- `internal/semantic/explanations.go` builds deterministic explanation sets with algorithm identity and read-only
  claim posture.
- `internal/semantic/explanations_test.go` covers source-track translation, ambiguous association anomaly
  explanation, and invariants for task prompt, output, source evidence, trajectory reference, algorithm, and claim
  posture.

## Not Claimed

- No hosted LLM service.
- No graph projector or SemStreams component.
- No COP UI semantic panel.
- No civilian advisory translation flow beyond the data contract needed to support it later.
- No SAPIENT, KLV, CS API, or standards conformance claim.
