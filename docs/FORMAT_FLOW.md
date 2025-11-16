# .flow Format

.flow is a UTF-8 text format described in `SYNTAX.md`. Each .flow file maps 1:1 to a JSON object. Parsers MUST:

- Replace TAB with 2 spaces
- Remove comments (text after `#`)
- Skip empty lines
- Compute indentation level = leading spaces / 2
- Maintain stack for nested objects
- Interpret values as string/number/boolean/array

Examples and guidelines are in `SYNTAX.md`.
