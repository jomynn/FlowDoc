# Multilanguage API (FlowDoc v1)

All language libraries implement the following API surface (naming adapted to language idioms):

## Core APIs

- `ParseFlow(text: string) -> object` — Parse .flow text into an object/dict/map.
- `StringifyFlow(obj: object) -> string` — Convert object to .flow text.
- `LoadFlow(path: string) -> object` — Read a .flow file and parse.
- `SaveFlow(path: string, obj: object)` — Serialize object to .flow and write to disk.
- `LoadFlowb(path: string) -> object` — Read .flowb (MessagePack) and decode.
- `SaveFlowb(path: string, obj: object)` — Encode object to MessagePack and write to disk.
- `ConvertFlowToJSON(flowText: string) -> string` — Convert .flow text to JSON string.
- `ConvertJSONToFlow(jsonText: string) -> string` — Convert JSON text to .flow text.

## Mapping Model APIs (Performance Feature)

The mapping model feature allows using short key aliases in data files while exposing full field names in code. See [docs/concepts/mapping-model.md](concepts/mapping-model.md) for details.

### Data Structures

**FieldDefinition:**
- `fullName: string` — Full field name
- `alias: string` — Short alias used in data
- `fieldType: string` — Type hint (string, int, float, bool, date, datetime)
- `fieldId: number` (optional) — Integer ID for binary format

**ModelDefinition:**
- `name: string` — Model name
- `fields: Map<string, FieldDefinition>` — Field definitions
- `aliasMap: Map<string, string>` — Alias to full name mapping

**ModelRegistry:**
- Container for multiple model definitions
- `registerModel(model)` — Add a model
- `getModel(name)` — Retrieve a model by name

### Parsing Functions

- `ParseFlowWithModel(text: string, registry?: ModelRegistry) -> object` — Parse .flow text with model support. If registry is not provided, attempts to extract `$models` from the text. Automatically expands aliases to full field names.

- `LoadFlowWithModel(path: string, registry?: ModelRegistry) -> object` — Read and parse a .flow file with model support.

### Language-Specific Examples

**Python:**
```python
from flowdoc import ParseFlowWithModel, ModelRegistry

# Automatic model extraction from file
result = ParseFlowWithModel(text)

# Or with explicit registry
registry = ModelRegistry()
# ... register models ...
result = ParseFlowWithModel(text, registry)
```

**Node/TypeScript:**
```typescript
import { parseFlowWithModel, ModelRegistry } from './flowdoc';

// Automatic model extraction
const result = parseFlowWithModel(text);

// Or with explicit registry
const registry = new ModelRegistry();
// ... register models ...
const result = parseFlowWithModel(text, registry);
```

**C#:**
```csharp
using FlowDoc;

// Automatic model extraction
var result = FlowDoc.FlowDoc.ParseFlowWithModel(text);

// Or with explicit registry
var registry = new ModelRegistry();
// ... register models ...
var result = FlowDoc.FlowDoc.ParseFlowWithModel(text, registry);
```

**Go:**
```go
import "flowdoc"

// Automatic model extraction
result, err := flowdoc.ParseFlowWithModel(text, nil)

// Or with explicit registry
registry := flowdoc.NewModelRegistry()
// ... register models ...
result, err := flowdoc.ParseFlowWithModel(text, registry)
```

**Rust:**
```rust
use flowdoc::{ParseFlowWithModel, ModelRegistry};

// Automatic model extraction
let result = ParseFlowWithModel(text, None);

// Or with explicit registry
let registry = ModelRegistry::new();
// ... register models ...
let result = ParseFlowWithModel(text, Some(&registry));
```

## General Notes

Implementations should try to keep types stable: maps/objects as dictionaries, arrays as language-native lists/arrays, basic types as strings, numbers, booleans.

When using mapping models, type hints provide additional type conversion during parsing.
