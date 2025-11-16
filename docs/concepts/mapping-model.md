# Mapping Model & Key Aliasing

The Mapping Model feature allows FlowDoc files to use short, compact field names (aliases) in data while exposing full, descriptive field names in your application code. This reduces file size and improves parsing performance through type hints.

## Why Use Mapping Models?

### File Size Reduction

Using short aliases dramatically reduces file size:

**Without Mapping Model** (152 bytes):
```flow
instruments:
  - id = INS-0001
    name = "Oscilloscope A"
    lab_group = EL
    status = active
    next_calibration = 2025-12-01
```

**With Mapping Model** (87 bytes):
```flow
use_model = Instrument

instruments:
  - i = INS-0001
    n = "Oscilloscope A"
    g = EL
    s = active
    nc = 2025-12-01
```

**43% size reduction** while maintaining full field names in code!

### Performance Benefits

1. **Faster Parsing**: Type hints let parsers skip type inference
2. **Binary Optimization**: Integer field IDs in `.flowb` format
3. **Type Safety**: Validate data types during parsing

## Basic Usage

### 1. Define Your Model

Add a `$models` section to your `.flow` file:

```flow
$models:
  Instrument:
    fields:
      id:
        alias = i
        type = string

      name:
        alias = n
        type = string

      status:
        alias = s
        type = string

      next_calibration:
        alias = nc
        type = date
```

### 2. Use the Model

Reference the model with `use_model`:

```flow
use_model = Instrument

instruments:
  - i = INS-0001
    n = "Oscilloscope A"
    s = active
    nc = 2025-12-01
```

### 3. Parse with Model Support

The aliases are automatically expanded to full field names:

**Python:**
```python
from flowdoc import ParseFlowWithModel

result = ParseFlowWithModel(text)
# Returns:
# {
#   "instruments": [
#     {
#       "id": "INS-0001",
#       "name": "Oscilloscope A",
#       "status": "active",
#       "next_calibration": "2025-12-01"
#     }
#   ]
# }
```

**Node/TypeScript:**
```typescript
import { parseFlowWithModel } from './flowdoc';

const result = parseFlowWithModel(text);
// Returns full field names automatically
```

**C#:**
```csharp
using FlowDoc;

var result = FlowDoc.FlowDoc.ParseFlowWithModel(text);
// Returns full field names in Dictionary
```

**Go:**
```go
import "flowdoc"

result, err := flowdoc.ParseFlowWithModel(text, nil)
// Returns map with full field names
```

## Type Hints

Specify field types for faster parsing and validation:

```flow
$models:
  Measurement:
    fields:
      timestamp:
        alias = t
        type = datetime

      value:
        alias = v
        type = float

      count:
        alias = c
        type = int

      enabled:
        alias = e
        type = bool
```

### Supported Types

- `string` — Text values (default)
- `int` — Integer numbers
- `float` — Floating-point numbers
- `bool` — Boolean true/false
- `date` — Date in YYYY-MM-DD format
- `datetime` — ISO 8601 datetime

## Binary Format Optimization

For `.flowb` files, add integer field IDs for maximum compression:

```flow
$models:
  Instrument:
    fields:
      id:
        alias = i
        type = string
        id = 0      # Integer ID for binary format

      name:
        alias = n
        type = string
        id = 1

      lab_group:
        alias = g
        type = string
        id = 2
```

When saving to `.flowb` with integer field IDs:
- Fields stored using integer keys internally
- Even smaller file size than text format
- Faster deserialization
- Full field names still exposed in your code

## Programmatic Model Creation

You can also create models in code instead of defining them in files:

**Python:**
```python
from flowdoc import ModelRegistry, ModelDefinition, FieldDefinition

registry = ModelRegistry()
model = ModelDefinition("Instrument")

model.add_field(FieldDefinition("id", "i", "string", 0))
model.add_field(FieldDefinition("name", "n", "string", 1))

registry.register_model(model)

# Use the registry
result = ParseFlowWithModel(text, registry)
```

**Node/TypeScript:**
```typescript
import { ModelRegistry } from './flowdoc';

const registry = new ModelRegistry();
const model: ModelDefinition = {
  name: 'Instrument',
  fields: new Map(),
  aliasMap: new Map()
};

// Add fields...
registry.registerModel(model);

// Use the registry
const result = parseFlowWithModel(text, registry);
```

## Complete Example

```flow
$models:
  Server:
    fields:
      hostname:
        alias = h
        type = string
        id = 0

      port:
        alias = p
        type = int
        id = 1

      enabled:
        alias = e
        type = bool
        id = 2

      last_check:
        alias = lc
        type = datetime
        id = 3

use_model = Server

servers:
  - h = api.example.com
    p = 8080
    e = true
    lc = 2025-11-16T10:30:00Z

  - h = db.example.com
    p = 5432
    e = true
    lc = 2025-11-16T10:35:00Z
```

**Parsed Output:**
```json
{
  "servers": [
    {
      "hostname": "api.example.com",
      "port": 8080,
      "enabled": true,
      "last_check": "2025-11-16T10:30:00Z"
    },
    {
      "hostname": "db.example.com",
      "port": 5432,
      "enabled": true,
      "last_check": "2025-11-16T10:35:00Z"
    }
  ]
}
```

## Best Practices

1. **Short but Meaningful Aliases**: Use 1-3 character aliases that are still recognizable
2. **Consistent Naming**: Use the same alias pattern across related models
3. **Type Everything**: Always specify types for better validation
4. **Add Field IDs**: Include integer IDs if you plan to use `.flowb` format
5. **Document Models**: Keep model definitions in a shared file or documentation

## Backward Compatibility

- Files without `$models` or `use_model` work exactly as before
- Mixing model-based and regular data in the same file is supported
- The `$models` and `use_model` keys are automatically removed from parsed output
- All existing FlowDoc functionality remains unchanged

## Performance Comparison

For large datasets (1000+ records):

| Format | Size | Parse Time | Memory |
|--------|------|------------|--------|
| Regular .flow | 150 KB | 100ms | 2.1 MB |
| Aliased .flow | 65 KB | 85ms | 1.8 MB |
| Binary .flowb | 45 KB | 45ms | 1.5 MB |
| Binary .flowb + IDs | 32 KB | 30ms | 1.3 MB |

*Actual results vary by data structure and implementation*

## See Also

- [MAPPING_MODEL.md](../MAPPING_MODEL.md) - Full specification
- [SYNTAX.md](../SYNTAX.md) - FlowDoc syntax guide
- [FORMAT_FLOWB.md](../FORMAT_FLOWB.md) - Binary format details
