# Multilanguage API (FlowDoc v1)

All language libraries implement the following API surface (naming adapted to language idioms):

- `ParseFlow(text: string) -> object` — Parse .flow text into an object/dict/map.
- `StringifyFlow(obj: object) -> string` — Convert object to .flow text.
- `LoadFlow(path: string) -> object` — Read a .flow file and parse.
- `SaveFlow(path: string, obj: object)` — Serialize object to .flow and write to disk.
- `LoadFlowb(path: string) -> object` — Read .flowb (MessagePack) and decode.
- `SaveFlowb(path: string, obj: object)` — Encode object to MessagePack and write to disk.
- `ConvertFlowToJSON(flowText: string) -> string` — Convert .flow text to JSON string.
- `ConvertJSONToFlow(jsonText: string) -> string` — Convert JSON text to .flow text.

Implementations should try to keep types stable: maps/objects as dictionaries, arrays as language-native lists/arrays, basic types as strings, numbers, booleans.
