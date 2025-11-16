"""
FlowDoc Python implementation

Implements ParseFlow, StringifyFlow, LoadFlow, SaveFlow, LoadFlowb, SaveFlowb,
ConvertFlowToJSON, ConvertJSONToFlow
"""
import re
import json
import msgpack
from typing import Any, Dict, List

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
