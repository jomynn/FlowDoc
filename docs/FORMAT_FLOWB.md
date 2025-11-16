# .flowb Format

.flowb is the binary equivalent of a .flow parsed object. It uses MessagePack encoding and decodes to the same object structure as `.flow`.

Requirements:
- Writers must encode the parsed object using MessagePack.
- Readers must decode MessagePack to native structures equivalent to the `.flow` representation.

This ensures full interchangeability between `.flow` and `.flowb`.
