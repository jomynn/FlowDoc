export type FlowValue = string | number | boolean | FlowObject | FlowValue[];
export type FlowObject = { [key: string]: FlowValue };

function tokenizeLines(text: string): string[] {
  return text.replace(/\t/g, '  ').split(/\r?\n/).map(l => {
    const noComment = l.split('#')[0];
    return noComment.replace(/\s+$/,'');
  }).filter(l => l.trim().length > 0);
}

function parseValue(raw: string): FlowValue {
  const v = raw.trim();
  if (v === 'true') return true;
  if (v === 'false') return false;
  if (/^".*"$/.test(v)) return v.slice(1, -1);
  if (/^\[.*\]$/.test(v)) {
    const inner = v.slice(1, -1).trim();
    if (inner === '') return [];
    return inner.split(',').map(x => parseValue(x));
  }
  if (!isNaN(Number(v))) {
    if (v.indexOf('.') >= 0) return Number(v);
    return parseInt(v, 10);
  }
  return v;
}

export function ParseFlow(text: string): FlowObject {
  const lines = tokenizeLines(text);
  const root: FlowObject = {};
  const stack: Array<{indent:number,node:FlowObject}> = [{indent:0,node:root}];

  for (const line of lines) {
    const leading = (line.match(/^\s*/)?.[0] ?? '').length;
    const indent = Math.floor(leading / 2);
    const trimmed = line.trim();
    if (trimmed.endsWith(':')) {
      const key = trimmed.slice(0, -1).trim();
      const obj: FlowObject = {};
      while (stack.length && stack[stack.length-1].indent >= indent) stack.pop();
      stack[stack.length-1].node[key] = obj;
      stack.push({indent: indent+1, node: obj});
    } else {
      const idx = trimmed.indexOf('=');
      if (idx < 0) continue;
      const key = trimmed.slice(0, idx).trim();
      const raw = trimmed.slice(idx+1).trim();
      while (stack.length && stack[stack.length-1].indent > indent) stack.pop();
      stack[stack.length-1].node[key] = parseValue(raw);
    }
  }
  return root;
}

function stringifyValue(v: FlowValue): string {
  if (typeof v === 'string') {
    if (/\s/.test(v) || v === '') return '"' + v + '"';
    return v;
  }
  if (typeof v === 'number' || typeof v === 'boolean') return String(v);
  if (Array.isArray(v)) return '[' + v.map(x => stringifyValue(x)).join(', ') + ']';
  if (typeof v === 'object') {
    // nested object: handled by parent stringify
    return '';
  }
  return String(v);
}

function stringifyObject(obj: FlowObject, indent: number): string {
  const lines: string[] = [];
  for (const k of Object.keys(obj)) {
    const v = obj[k];
    if (v && typeof v === 'object' && !Array.isArray(v)) {
      lines.push(' '.repeat(indent) + k + ':');
      lines.push(stringifyObject(v as FlowObject, indent + 2));
    } else {
      lines.push(' '.repeat(indent) + k + ' = ' + stringifyValue(v));
    }
  }
  return lines.join('\n');
}

export function StringifyFlow(obj: FlowObject): string {
  return stringifyObject(obj, 0) + '\n';
}

export function ConvertFlowToJSON(flowText: string): string {
  return JSON.stringify(ParseFlow(flowText), null, 2);
}

export function ConvertJSONToFlow(jsonText: string): string {
  const obj = JSON.parse(jsonText) as FlowObject;
  return StringifyFlow(obj);
}
