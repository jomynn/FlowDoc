# Parsing Rules

Universal parser algorithm for FlowDoc:

1. Split input into lines (by `\n`).
2. Replace tabs with two spaces.
3. Remove comments (from `#` to end of line).
4. Skip empty lines.
5. For each remaining line, count leading spaces and compute `indent = leading_spaces / 2` (integer division).
6. Maintain a stack of objects and their indent levels. Root is indent 0.
7. If line ends with `:` then the line defines an object key. Create a new map at that key and push it to the stack at indent level.
8. Otherwise parse `key = value` where `value` may be:
   - Quoted string: `"..."`
   - Number (integer or float)
   - Boolean: `true` or `false`
   - Array: `[a, b, c]` (comma separated). Elements parsed by same rules.
   - Raw unquoted string fallback

9. When indentation decreases, pop stack until matching indent level.

Edge cases:
- Lines with malformed indentation or syntax should throw/return parse error where possible.
- Empty arrays `[]` are allowed.
