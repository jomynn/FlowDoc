const fs = require('fs');
const msgpack = require('@msgpack/msgpack');

function _tokenizeLines(text) {
  return text.replace(/\t/g, '  ').split(/\r?\n/).map(l => {
    const noComment = l.split('#')[0];
    return noComment.replace(/\s+$/,'');
  }).filter(l => l.trim().length > 0);
}

function _parseValue(raw) {
  const v = raw.trim();
  if (v === 'true') return true;
  if (v === 'false') return false;
  if (/^".*"$/.test(v)) return v.slice(1, -1);
  if (/^\[.*\]$/.test(v)) {
    const inner = v.slice(1, -1).trim();
    if (inner === '') return [];
    return inner.split(',').map(x => _parseValue(x));
  }
  if (!isNaN(Number(v))) {
    if (v.indexOf('.') >= 0) return Number(v);
    return parseInt(v, 10);
  }
  return v;
}

function ParseFlow(text) {
  const lines = _tokenizeLines(text);
  const root = {};
  const stack = [{indent:0, node: root}];

  for (const line of lines) {
    const leading = line.match(/^\s*/)[0].length;
    const indent = Math.floor(leading / 2);
    const trimmed = line.trim();
    if (trimmed.endsWith(':')) {
      const key = trimmed.slice(0, -1).trim();
      const obj = {};
      while (stack.length && stack[stack.length-1].indent >= indent) stack.pop();
      stack[stack.length-1].node[key] = obj;
      stack.push({indent: indent+1, node: obj});
    } else {
      const parts = trimmed.split('=');
      if (parts.length < 2) continue;
      const key = parts[0].trim();
      const raw = parts.slice(1).join('=').trim();
      while (stack.length && stack[stack.length-1].indent > indent) stack.pop();
      stack[stack.length-1].node[key] = _parseValue(raw);
    }
  }
  return root;
}

function _stringifyValue(v, indent) {
  if (typeof v === 'string') {
    if (/\s/.test(v) || v === '') return '"' + v + '"';
    return v;
  }
  if (typeof v === 'number' || typeof v === 'boolean') return String(v);
  if (Array.isArray(v)) return '[' + v.map(x => _stringifyValue(x, indent)).join(', ') + ']';
  if (v && typeof v === 'object') {
    const lines = [];
    for (const k of Object.keys(v)) {
      const val = v[k];
      if (val && typeof val === 'object' && !Array.isArray(val)) {
        lines.push(''.padStart(indent, ' ') + k + ':');
        lines.push(_stringifyObject(val, indent + 2));
      } else {
        lines.push(''.padStart(indent, ' ') + k + ' = ' + _stringifyValue(val, indent));
      }
    }
    return lines.join('\n');
  }
  return String(v);
}

function _stringifyObject(obj, indent) {
  const lines = [];
  for (const k of Object.keys(obj)) {
    const v = obj[k];
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      lines.push(''.padStart(indent, ' ') + k + ':');
      lines.push(_stringifyObject(v, indent + 2));
    } else {
      lines.push(''.padStart(indent, ' ') + k + ' = ' + _stringifyValue(v, indent));
    }
  }
  return lines.join('\n');
}

function StringifyFlow(obj) {
  return _stringifyObject(obj, 0) + '\n';
}

function LoadFlow(path) {
  const text = fs.readFileSync(path, 'utf8');
  return ParseFlow(text);
}

function SaveFlow(path, obj) {
  fs.writeFileSync(path, StringifyFlow(obj), 'utf8');
}

function LoadFlowb(path) {
  const buf = fs.readFileSync(path);
  return msgpack.decode(buf);
}

function SaveFlowb(path, obj) {
  const buf = msgpack.encode(obj);
  fs.writeFileSync(path, Buffer.from(buf));
}

function ConvertFlowToJSON(flowText) {
  return JSON.stringify(ParseFlow(flowText), null, 2);
}

function ConvertJSONToFlow(jsonText) {
  const obj = JSON.parse(jsonText);
  return StringifyFlow(obj);
}

module.exports = {
  ParseFlow,
  StringifyFlow,
  LoadFlow,
  SaveFlow,
  LoadFlowb,
  SaveFlowb,
  ConvertFlowToJSON,
  ConvertJSONToFlow
};
