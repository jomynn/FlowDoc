"""
FlowDoc Python implementation

Implements ParseFlow, StringifyFlow, LoadFlow, SaveFlow, LoadFlowb, SaveFlowb,
ConvertFlowToJSON, ConvertJSONToFlow, and mapping model support
"""
import re
import json
import msgpack
from typing import Any, Dict, List, Optional
from datetime import datetime, date

# ============================================
# Mapping Model Support
# ============================================

class FieldDefinition:
    """Definition for a single field in a model"""
    def __init__(self, full_name: str, alias: str, field_type: str = "string", field_id: Optional[int] = None):
        self.full_name = full_name
        self.alias = alias
        self.field_type = field_type
        self.field_id = field_id

class ModelDefinition:
    """Definition for a complete model"""
    def __init__(self, name: str):
        self.name = name
        self.fields: Dict[str, FieldDefinition] = {}  # indexed by full name
        self.alias_map: Dict[str, str] = {}  # alias -> full name

    def add_field(self, field: FieldDefinition):
        self.fields[field.full_name] = field
        self.alias_map[field.alias] = field.full_name

class ModelRegistry:
    """Registry containing multiple model definitions"""
    def __init__(self):
        self.models: Dict[str, ModelDefinition] = {}

    def register_model(self, model: ModelDefinition):
        self.models[model.name] = model

    def get_model(self, name: str) -> Optional[ModelDefinition]:
        return self.models.get(name)

def _parse_typed_value(raw: str, field_type: str) -> Any:
    """Parse value with type hint"""
    v = raw.strip()

    if field_type == "bool":
        if v == "true":
            return True
        if v == "false":
            return False
        raise ValueError(f"Invalid boolean value: {v}")

    if field_type == "int":
        return int(v)

    if field_type == "float":
        return float(v)

    if field_type == "date":
        # Parse YYYY-MM-DD format
        if re.match(r'^\d{4}-\d{2}-\d{2}$', v):
            return v  # Keep as string for JSON compatibility
        raise ValueError(f"Invalid date format (expected YYYY-MM-DD): {v}")

    if field_type == "datetime":
        # Parse ISO 8601 datetime
        if re.match(r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}', v):
            return v  # Keep as string for JSON compatibility
        raise ValueError(f"Invalid datetime format (expected ISO 8601): {v}")

    # string type - remove quotes if present
    if v.startswith('"') and v.endswith('"'):
        return v[1:-1]

    return v

def _extract_models_from_dict(data: Dict[str, Any]) -> Optional[ModelRegistry]:
    """Extract $models from parsed dictionary"""
    if "$models" not in data:
        return None

    registry = ModelRegistry()
    models_data = data["$models"]

    if not isinstance(models_data, dict):
        return None

    for model_name, model_spec in models_data.items():
        if not isinstance(model_spec, dict) or "fields" not in model_spec:
            continue

        model_def = ModelDefinition(model_name)
        fields_data = model_spec["fields"]

        if not isinstance(fields_data, dict):
            continue

        for full_name, field_spec in fields_data.items():
            if not isinstance(field_spec, dict):
                continue

            alias = field_spec.get("alias", full_name)
            field_type = field_spec.get("type", "string")
            field_id = field_spec.get("id")

            field_def = FieldDefinition(full_name, alias, field_type, field_id)
            model_def.add_field(field_def)

        registry.register_model(model_def)

    return registry

def _apply_model_to_dict(data: Dict[str, Any], model: ModelDefinition) -> Dict[str, Any]:
    """Apply model to expand aliases to full field names"""
    result = {}

    for key, value in data.items():
        # Check if this key is an alias
        full_name = model.alias_map.get(key, key)
        field_def = model.fields.get(full_name)

        # Process value
        if isinstance(value, dict):
            # Recursively apply model to nested dicts
            result[full_name] = _apply_model_to_dict(value, model)
        elif isinstance(value, list):
            # Apply model to list items that are dicts
            result[full_name] = [
                _apply_model_to_dict(item, model) if isinstance(item, dict) else item
                for item in value
            ]
        else:
            # Apply type conversion if field definition exists
            if field_def:
                try:
                    result[full_name] = _parse_typed_value(str(value), field_def.field_type)
                except (ValueError, TypeError):
                    result[full_name] = value
            else:
                result[full_name] = value

    return result

# ============================================
# Core Parsing Functions
# ============================================

def _tokenize_lines(text: str) -> List[str]:
    text = text.replace('\t', '  ')
    lines = []
    for line in text.splitlines():
        no_comment = line.split('#', 1)[0]
        if no_comment.strip():
            lines.append(no_comment.rstrip())
    return lines

def _parse_value(raw: str) -> Any:
    v = raw.strip()
    if v == 'true':
        return True
    if v == 'false':
        return False
    if re.match(r'^".*"$', v):
        return v[1:-1]
    if re.match(r'^\[.*\]$', v):
        inner = v[1:-1].strip()
        if inner == '':
            return []
        parts = [p.strip() for p in inner.split(',')]
        return [_parse_value(p) for p in parts]
    try:
        if '.' in v:
            return float(v)
        return int(v)
    except Exception:
        return v

def ParseFlow(text: str) -> Dict[str, Any]:
    lines = _tokenize_lines(text)
    root: Dict[str, Any] = {}
    stack = [(0, root)]
    for line in lines:
        leading = len(re.match(r'^\s*', line).group(0))
        indent = leading // 2
        trimmed = line.strip()
        if trimmed.endswith(':'):
            key = trimmed[:-1].strip()
            obj: Dict[str, Any] = {}
            while stack and stack[-1][0] >= indent:
                stack.pop()
            stack[-1][1][key] = obj
            stack.append((indent+1, obj))
        else:
            if '=' not in trimmed:
                continue
            key, raw = trimmed.split('=', 1)
            key = key.strip()
            raw = raw.strip()
            while stack and stack[-1][0] > indent:
                stack.pop()
            stack[-1][1][key] = _parse_value(raw)
    return root

def _stringify_value(v: Any) -> str:
    if isinstance(v, str):
        if re.search(r'\s', v) or v == '':
            return '"' + v + '"'
        return v
    if isinstance(v, bool):
        return 'true' if v else 'false'
    if isinstance(v, (int, float)):
        return str(v)
    if isinstance(v, list):
        return '[' + ', '.join(_stringify_value(x) for x in v) + ']'
    return ''

def _stringify_object(obj: Dict[str, Any], indent: int = 0) -> str:
    lines: List[str] = []
    pad = ' ' * indent
    for k, v in obj.items():
        if isinstance(v, dict):
            lines.append(f"{pad}{k}:")
            lines.append(_stringify_object(v, indent + 2))
        else:
            lines.append(f"{pad}{k} = {_stringify_value(v)}")
    return '\n'.join(lines)

def StringifyFlow(obj: Dict[str, Any]) -> str:
    return _stringify_object(obj, 0) + '\n'

def LoadFlow(path: str) -> Dict[str, Any]:
    with open(path, 'r', encoding='utf8') as f:
        return ParseFlow(f.read())

def SaveFlow(path: str, obj: Dict[str, Any]):
    with open(path, 'w', encoding='utf8') as f:
        f.write(StringifyFlow(obj))

def LoadFlowb(path: str) -> Dict[str, Any]:
    with open(path, 'rb') as f:
        data = f.read()
    return msgpack.unpackb(data, raw=False)

def SaveFlowb(path: str, obj: Dict[str, Any]):
    data = msgpack.packb(obj, use_bin_type=True)
    with open(path, 'wb') as f:
        f.write(data)

def ConvertFlowToJSON(flowText: str) -> str:
    return json.dumps(ParseFlow(flowText), indent=2)

def ConvertJSONToFlow(jsonText: str) -> str:
    obj = json.loads(jsonText)
    return StringifyFlow(obj)

def ParseFlowWithModel(text: str, registry: Optional[ModelRegistry] = None) -> Dict[str, Any]:
    """
    Parse FlowDoc text with optional model registry.
    If registry is None, attempts to extract $models from the text.
    Applies model transformation if use_model directive is found.
    """
    # First, parse normally
    data = ParseFlow(text)

    # Extract or use provided registry
    if registry is None:
        registry = _extract_models_from_dict(data)

    # Check for use_model directive
    if registry and "use_model" in data:
        model_name = data["use_model"]
        model = registry.get_model(model_name)

        if model is None:
            raise ValueError(f"Model '{model_name}' not found in registry")

        # Remove use_model and $models from result
        result = {k: v for k, v in data.items() if k not in ["use_model", "$models"]}

        # Apply model to all top-level values
        final_result = {}
        for key, value in result.items():
            if isinstance(value, dict):
                final_result[key] = _apply_model_to_dict(value, model)
            elif isinstance(value, list):
                final_result[key] = [
                    _apply_model_to_dict(item, model) if isinstance(item, dict) else item
                    for item in value
                ]
            else:
                final_result[key] = value

        return final_result

    # No model application needed
    # Remove $models from output if present
    if "$models" in data:
        return {k: v for k, v in data.items() if k != "$models"}

    return data

def LoadFlowWithModel(path: str, registry: Optional[ModelRegistry] = None) -> Dict[str, Any]:
    """Load and parse a .flow file with model support"""
    with open(path, 'r', encoding='utf8') as f:
        return ParseFlowWithModel(f.read(), registry)

if __name__ == '__main__':
    # quick self-test
    sample = """
app:
  name = TestApp
  version = 1.2.3
features:
  list = [a, b, c]
  enabled = true
"""
    print(ConvertFlowToJSON(sample))
