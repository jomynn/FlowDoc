# FlowDoc Mapping Model Specification

## Overview

The Mapping Model feature allows FlowDoc files to use short keys (aliases) in data while maintaining full field names in the application layer. This improves file size and parsing performance through type hints and optional integer field IDs for binary format.

## Goals

1. **Size Reduction**: Use short aliases instead of full field names in `.flow` files
2. **Performance**: Provide type hints to accelerate parsing
3. **Binary Optimization**: Allow integer field IDs in `.flowb` for minimal size
4. **Developer Experience**: Expose full field names in parsed objects

## Model Definition Syntax

### Reserved Key: `$models`

Models are defined using the top-level reserved key `$models`:

```flow
$models:
  ModelName:
    fields:
      full_field_name:
        alias = short_alias
        type = field_type
        id = integer_id  # optional, for .flowb optimization
```

### Field Types

Supported type hints:
- `string` — Text values
- `int` — Integer numbers
- `float` — Floating-point numbers
- `bool` — Boolean true/false
- `date` — Date in YYYY-MM-DD format
- `datetime` — ISO 8601 datetime

### Complete Example

```flow
$models:
  Instrument:
    fields:
      id:
        alias = i
        type = string
        id = 0

      name:
        alias = n
        type = string
        id = 1

      lab_group:
        alias = g
        type = string
        id = 2

      status:
        alias = s
        type = string
        id = 3

      next_calibration:
        alias = nc
        type = date
        id = 4
```

## Using Models in Data Files

### Declaring Model Usage

Use the `use_model` directive to apply a model:

```flow
use_model = Instrument

instruments:
  - i = INS-0001
    n = "Oscilloscope A"
    g = EL
    s = active
    nc = 2025-12-01

  - i = INS-0002
    n = "Micrometer B"
    g = DM
    s = active
    nc = 2025-09-15
```

### Parsed Output

When parsed with the model, short aliases are expanded to full field names:

```json
{
  "instruments": [
    {
      "id": "INS-0001",
      "name": "Oscilloscope A",
      "lab_group": "EL",
      "status": "active",
      "next_calibration": "2025-12-01"
    },
    {
      "id": "INS-0002",
      "name": "Micrometer B",
      "lab_group": "DM",
      "status": "active",
      "next_calibration": "2025-09-15"
    }
  ]
}
```

## Binary Format (.flowb) Optimization

For `.flowb` files, when integer field IDs are defined in the model:

1. **Internal Storage**: Fields are stored using integer keys instead of strings
2. **Parsing**: SDK uses integer lookups for faster deserialization
3. **Exposure**: Applications still receive objects with full field names

This provides maximum size reduction and parsing speed for binary files.

## Library Implementation Requirements

### Core Data Structures

Each SDK must implement:

1. **FieldDefinition**
   - `alias: string` — Short key used in data
   - `type: string` — Type hint for parsing
   - `id: number` (optional) — Integer ID for binary format

2. **ModelDefinition**
   - `name: string` — Model name
   - `fields: Map<string, FieldDefinition>` — Field definitions indexed by full name

3. **ModelRegistry**
   - Container for multiple model definitions
   - Methods to register and retrieve models
   - Can be loaded from `$models` in a file or built programmatically

### Required APIs

Each language SDK must provide:

#### Parse with Model
- **C#**: `ParseFlowWithModel(string text, ModelRegistry models)`
- **Node/TS**: `parseFlowWithModel(text: string, models: ModelRegistry): object`
- **Python**: `parse_flow_with_model(text: str, models: ModelRegistry) -> dict`
- **Go**: `ParseFlowWithModel(text string, models *ModelRegistry) (map[string]interface{}, error)`
- **Rust**: `parse_flow_with_model(text: &str, models: &ModelRegistry) -> Value`

#### Model Registry Construction
- **From File**: Extract `$models` from a `.flow` file
- **Programmatic**: Build models in code

#### Binary Optimization
- For `.flowb`, support integer field IDs when available
- Fallback to string keys when IDs not defined

### Parsing Behavior

1. **Model Extraction**: If `$models` exists, extract and register models before parsing data
2. **Model Application**: If `use_model` directive found, apply specified model to transform aliases
3. **Type Conversion**: Use type hints to parse values correctly (e.g., "2025-12-01" as Date object)
4. **Field Expansion**: Replace alias keys with full field names in output
5. **Nested Objects**: Apply model recursively to nested structures and arrays

## Type Conversion Rules

When type hints are provided:

- **string**: Keep as string (default)
- **int**: Parse to integer number
- **float**: Parse to floating-point number
- **bool**: Parse `true`/`false` to boolean
- **date**: Parse YYYY-MM-DD to Date object (language-specific)
- **datetime**: Parse ISO 8601 to DateTime object (language-specific)

## Example Use Cases

### Configuration Files

Reduce verbose configuration files:

```flow
$models:
  ServerConfig:
    fields:
      hostname:
        alias = h
        type = string
      port:
        alias = p
        type = int
      enabled:
        alias = e
        type = bool

use_model = ServerConfig

servers:
  - h = api.example.com
    p = 8080
    e = true
  - h = db.example.com
    p = 5432
    e = true
```

### Data Export

Compact data serialization:

```flow
$models:
  Measurement:
    fields:
      timestamp:
        alias = t
        type = datetime
        id = 0
      value:
        alias = v
        type = float
        id = 1
      unit:
        alias = u
        type = string
        id = 2

use_model = Measurement

data:
  - t = 2025-11-16T10:30:00Z
    v = 23.5
    u = celsius
  - t = 2025-11-16T10:31:00Z
    v = 23.7
    u = celsius
```

## Compatibility

- **Backward Compatible**: Files without `$models` or `use_model` parse normally
- **Mixed Usage**: Can have both model-based and regular data in same file
- **Validation**: Libraries should validate that referenced models exist
- **Error Handling**: Clear errors when model not found or field types mismatch
