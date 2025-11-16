# FlowDoc v1

FlowDoc is a small, human-friendly data format and libraries for multiple languages. See `docs/` for full specification and usage.

This repository contains implementations for Node.js, Next.js (TypeScript), Python, Go, Rust, C#, and optional PHP, plus documentation and samples.

## Features

- **Clean Syntax**: Human-readable, minimal punctuation
- **Small File Size**: Compact format, smaller than JSON/YAML
- **Binary Format**: `.flowb` uses MessagePack for even smaller size
- **Mapping Model**: Use short key aliases for maximum compression while keeping full field names in code (NEW!)
- **Type Hints**: Optional type annotations for faster parsing
- **Multi-Language**: Consistent API across Python, Node.js, C#, Go, Rust

## Mapping Model & Key Aliasing (Performance Feature)

The mapping model feature allows you to use short aliases in your data files while exposing full, descriptive field names in your application code.

**Example - Before (152 bytes):**
```flow
instruments:
  - id = INS-0001
    name = "Oscilloscope A"
    lab_group = EL
    status = active
```

**Example - After (87 bytes):**
```flow
$models:
  Instrument:
    fields:
      id: { alias = i, type = string }
      name: { alias = n, type = string }
      lab_group: { alias = g, type = string }
      status: { alias = s, type = string }

use_model = Instrument

instruments:
  - i = INS-0001
    n = "Oscilloscope A"
    g = EL
    s = active
```

**43% size reduction!** Your code still receives full field names like `id`, `name`, etc.

See [docs/concepts/mapping-model.md](docs/concepts/mapping-model.md) for details.

# FlowDoc
FlowDoc v1 Multilanguage Library
