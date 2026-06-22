# KLV Fixtures

This directory holds small SemOps-owned KLV fixture inputs.

`misb0601-truth.json` is a deterministic MISB ST 0601 subset truth fixture. Tests encode it into a KLV local-set
packet at runtime, decode that packet through the SemOps KLV component package, and compare decoded values back to the
truth within MISB integer quantization tolerances.

This fixture is storage, governance, and parser engineering evidence. It is not a public-sample smoke test, live media
test, STANAG 4609 conformance result, or official certification artifact.
