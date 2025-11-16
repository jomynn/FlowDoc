# FlowDoc Test Runner

This file contains instructions and test cases for validating FlowDoc implementations.

Tests to run (per-language):

1. Parsing: parse `tests/test.flow` and compare structure to `tests/test.json`.
2. Stringifying: parse then stringify and ensure text matches expected structure.
3. Binary read/write: SaveFlowb then LoadFlowb and compare objects.
4. JSON ↔ FLOW conversion: ConvertFlowToJSON and ConvertJSONToFlow round-trip.
5. Deep nesting: add nested structures and verify.
6. Arrays and type detection.

Use language-specific test harnesses or manual checks.
