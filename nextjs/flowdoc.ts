export type FlowValue = string | number | boolean | FlowObject | FlowValue[];
export type FlowObject = { [key: string]: FlowValue };

// ============================================
// Mapping Model Support
// ============================================

export interface FieldDefinition {
  fullName: string;
  alias: string;
  fieldType: string;
  fieldId?: number;
}

export interface ModelDefinition {
  name: string;
  fields: Map<string, FieldDefinition>;  // indexed by full name
  aliasMap: Map<string, string>;  // alias -> full name
}

export class ModelRegistry {
  private models: Map<string, ModelDefinition> = new Map();

  registerModel(model: ModelDefinition): void {
    this.models.set(model.name, model);
  }

  getModel(name: string): ModelDefinition | undefined {
    return this.models.get(name);
  }
}

function parseTypedValue(raw: string, fieldType: string): FlowValue {
  const v = raw.trim();

  if (fieldType === 'bool') {
    if (v === 'true') return true;
    if (v === 'false') return false;
    throw new Error(`Invalid boolean value: ${v}`);
  }

  if (fieldType === 'int') {
    const num = parseInt(v, 10);
    if (isNaN(num)) throw new Error(`Invalid integer value: ${v}`);
    return num;
  }

  if (fieldType === 'float') {
    const num = parseFloat(v);
    if (isNaN(num)) throw new Error(`Invalid float value: ${v}`);
    return num;
  }

  if (fieldType === 'date') {
    // Validate YYYY-MM-DD format
    if (!/^\d{4}-\d{2}-\d{2}$/.test(v)) {
      throw new Error(`Invalid date format (expected YYYY-MM-DD): ${v}`);
    }
    return v;  // Keep as string for JSON compatibility
  }

  if (fieldType === 'datetime') {
    // Validate ISO 8601 datetime
    if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(v)) {
      throw new Error(`Invalid datetime format (expected ISO 8601): ${v}`);
    }
    return v;  // Keep as string for JSON compatibility
  }

  // string type - remove quotes if present
  if (v.startsWith('"') && v.endsWith('"')) {
    return v.slice(1, -1);
  }

  return v;
}

function extractModelsFromObject(data: FlowObject): ModelRegistry | null {
  if (!('$models' in data)) return null;

  const registry = new ModelRegistry();
  const modelsData = data['$models'];

  if (typeof modelsData !== 'object' || Array.isArray(modelsData)) {
    return null;
  }

  const models = modelsData as FlowObject;

  for (const modelName of Object.keys(models)) {
    const modelSpec = models[modelName];
    if (typeof modelSpec !== 'object' || Array.isArray(modelSpec)) continue;

    const spec = modelSpec as FlowObject;
    if (!('fields' in spec)) continue;

    const modelDef: ModelDefinition = {
      name: modelName,
      fields: new Map(),
      aliasMap: new Map()
    };

    const fieldsData = spec.fields;
    if (typeof fieldsData !== 'object' || Array.isArray(fieldsData)) continue;

    const fields = fieldsData as FlowObject;

    for (const fullName of Object.keys(fields)) {
      const fieldSpec = fields[fullName];
      if (typeof fieldSpec !== 'object' || Array.isArray(fieldSpec)) continue;

      const fspec = fieldSpec as FlowObject;
      const alias = (fspec.alias as string) || fullName;
      const fieldType = (fspec.type as string) || 'string';
      const fieldId = fspec.id as number | undefined;

      const fieldDef: FieldDefinition = {
        fullName,
        alias,
        fieldType,
        fieldId
      };

      modelDef.fields.set(fullName, fieldDef);
      modelDef.aliasMap.set(alias, fullName);
    }

    registry.registerModel(modelDef);
  }

  return registry;
}

function applyModelToObject(data: FlowObject, model: ModelDefinition): FlowObject {
  const result: FlowObject = {};

  for (const key of Object.keys(data)) {
    // Check if this key is an alias
    const fullName = model.aliasMap.get(key) || key;
    const fieldDef = model.fields.get(fullName);

    const value = data[key];

    // Process value
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      // Recursively apply model to nested objects
      result[fullName] = applyModelToObject(value as FlowObject, model);
    } else if (Array.isArray(value)) {
      // Apply model to list items that are objects
      result[fullName] = value.map(item =>
        item && typeof item === 'object' && !Array.isArray(item)
          ? applyModelToObject(item as FlowObject, model)
          : item
      );
    } else {
      // Apply type conversion if field definition exists
      if (fieldDef) {
        try {
          result[fullName] = parseTypedValue(String(value), fieldDef.fieldType);
        } catch {
          result[fullName] = value;
        }
      } else {
        result[fullName] = value;
      }
    }
  }

  return result;
}

// ============================================
// Core Parsing Functions
// ============================================

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

export function parseFlowWithModel(text: string, registry?: ModelRegistry): FlowObject {
  // First, parse normally
  const data = ParseFlow(text);

  // Extract or use provided registry
  let modelRegistry = registry;
  if (!modelRegistry) {
    modelRegistry = extractModelsFromObject(data) || undefined;
  }

  // Check for use_model directive
  if (modelRegistry && 'use_model' in data) {
    const modelName = data['use_model'] as string;
    const model = modelRegistry.getModel(modelName);

    if (!model) {
      throw new Error(`Model '${modelName}' not found in registry`);
    }

    // Remove use_model and $models from result
    const result: FlowObject = {};
    for (const key of Object.keys(data)) {
      if (key !== 'use_model' && key !== '$models') {
        result[key] = data[key];
      }
    }

    // Apply model to all top-level values
    const finalResult: FlowObject = {};
    for (const key of Object.keys(result)) {
      const value = result[key];
      if (value && typeof value === 'object' && !Array.isArray(value)) {
        finalResult[key] = applyModelToObject(value as FlowObject, model);
      } else if (Array.isArray(value)) {
        finalResult[key] = value.map(item =>
          item && typeof item === 'object' && !Array.isArray(item)
            ? applyModelToObject(item as FlowObject, model)
            : item
        );
      } else {
        finalResult[key] = value;
      }
    }

    return finalResult;
  }

  // No model application needed
  // Remove $models from output if present
  if ('$models' in data) {
    const result: FlowObject = {};
    for (const key of Object.keys(data)) {
      if (key !== '$models') {
        result[key] = data[key];
      }
    }
    return result;
  }

  return data;
}
