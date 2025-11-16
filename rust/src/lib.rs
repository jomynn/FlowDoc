use serde_json::{Value, Map};
use std::collections::HashMap;
use std::fs;

// ============================================
// Mapping Model Support
// ============================================

pub struct FieldDefinition {
    pub full_name: String,
    pub alias: String,
    pub field_type: String,
    pub field_id: Option<i64>,
}

pub struct ModelDefinition {
    pub name: String,
    pub fields: HashMap<String, FieldDefinition>,
    pub alias_map: HashMap<String, String>,
}

impl ModelDefinition {
    pub fn new(name: String) -> Self {
        ModelDefinition {
            name,
            fields: HashMap::new(),
            alias_map: HashMap::new(),
        }
    }

    pub fn add_field(&mut self, field: FieldDefinition) {
        self.alias_map.insert(field.alias.clone(), field.full_name.clone());
        self.fields.insert(field.full_name.clone(), field);
    }
}

pub struct ModelRegistry {
    models: HashMap<String, ModelDefinition>,
}

impl ModelRegistry {
    pub fn new() -> Self {
        ModelRegistry {
            models: HashMap::new(),
        }
    }

    pub fn register_model(&mut self, model: ModelDefinition) {
        self.models.insert(model.name.clone(), model);
    }

    pub fn get_model(&self, name: &str) -> Option<&ModelDefinition> {
        self.models.get(name)
    }
}

// ============================================
// Core Parsing Functions
// ============================================

fn tokenize_lines(text: &str) -> Vec<String> {
    text.replace("\t", "  ").lines().map(|l| {
        let no_comment = l.split('#').next().unwrap_or("");
        no_comment.trim_end().to_string()
    }).filter(|l| !l.trim().is_empty()).collect()
}

fn parse_value(raw: &str) -> Value {
    let v = raw.trim();
    if v == "true" { return Value::Bool(true); }
    if v == "false" { return Value::Bool(false); }
    if v.starts_with('"') && v.ends_with('"') {
        return Value::String(v[1..v.len()-1].to_string());
    }
    if v.starts_with('[') && v.ends_with(']') {
        let inner = v[1..v.len()-1].trim();
        if inner.is_empty() { return Value::Array(vec![]); }
        let elems = inner.split(',').map(|s| parse_value(s)).collect();
        return Value::Array(elems);
    }
    if let Ok(i) = v.parse::<i64>() { return Value::Number(i.into()); }
    if let Ok(f) = v.parse::<f64>() { return serde_json::Number::from_f64(f).map(Value::Number).unwrap_or(Value::String(v.to_string())); }
    Value::String(v.to_string())
}

pub fn ParseFlow(text: &str) -> Value {
    let lines = tokenize_lines(text);
    let mut root = Map::new();
    let mut stack: Vec<(usize, Map<String, Value>)> = vec![(0, Map::new())];
    for line in lines {
        let leading = line.chars().take_while(|c| c.is_whitespace()).count();
        let indent = leading / 2;
        let trimmed = line.trim();
        if trimmed.ends_with(':') {
            let key = trimmed[..trimmed.len()-1].trim();
            let obj = Map::new();
            while stack.last().map(|(i, _)| *i).unwrap_or(0) >= indent {
                stack.pop();
            }
            if let Some((_, ref mut parent)) = stack.last_mut() {
                parent.insert(key.to_string(), Value::Object(obj.clone()));
                stack.push((indent+1, obj));
            }
        } else {
            if let Some(pos) = trimmed.find('=') {
                let key = trimmed[..pos].trim();
                let raw = trimmed[pos+1..].trim();
                while stack.last().map(|(i, _)| *i).unwrap_or(0) > indent { stack.pop(); }
                if let Some((_, ref mut parent)) = stack.last_mut() {
                    parent.insert(key.to_string(), parse_value(raw));
                }
            }
        }
    }
    // reconstruct root from stack[0]
    if let Some((_, m)) = stack.into_iter().next() { Value::Object(m) } else { Value::Object(root) }
}

pub fn StringifyFlow(val: &Value) -> String {
    fn write_obj(map: &Map<String, Value>, indent: usize, out: &mut String) {
        let pad = " ".repeat(indent);
        for (k, v) in map {
            match v {
                Value::Object(m) => {
                    out.push_str(&format!("{}{}:\n", pad, k));
                    write_obj(m, indent+2, out);
                }
                Value::Array(arr) => {
                    let parts: Vec<String> = arr.iter().map(|e| match e {
                        Value::String(s) => if s.contains(' ') { format!("\"{}\"", s) } else { s.clone() },
                        Value::Bool(b) => b.to_string(),
                        Value::Number(n) => n.to_string(),
                        _ => format!("{}", e)
                    }).collect();
                    out.push_str(&format!("{}{} = [{}]\n", pad, k, parts.join(", ")));
                }
                Value::String(s) => {
                    if s.contains(' ') {
                        out.push_str(&format!("{}{} = \"{}\"\n", pad, k, s));
                    } else {
                        out.push_str(&format!("{}{} = {}\n", pad, k, s));
                    }
                }
                Value::Bool(b) => out.push_str(&format!("{}{} = {}\n", pad, k, b)),
                Value::Number(n) => out.push_str(&format!("{}{} = {}\n", pad, k, n)),
                _ => {}
            }
        }
    }
    if let Value::Object(m) = val { let mut out = String::new(); write_obj(m, 0, &mut out); out } else { String::new() }
}

pub fn LoadFlow(path: &str) -> Result<Value, std::io::Error> {
    let s = fs::read_to_string(path)?;
    Ok(ParseFlow(&s))
}

pub fn SaveFlow(path: &str, val: &Value) -> Result<(), std::io::Error> {
    fs::write(path, StringifyFlow(val))
}

pub fn LoadFlowb(path: &str) -> Result<Value, Box<dyn std::error::Error>> {
    let data = fs::read(path)?;
    let v: Value = rmp_serde::from_slice(&data)?;
    Ok(v)
}

pub fn SaveFlowb(path: &str, val: &Value) -> Result<(), Box<dyn std::error::Error>> {
    let buf = rmp_serde::to_vec(val)?;
    fs::write(path, buf)?;
    Ok(())
}

pub fn ConvertFlowToJSON(flowText: &str) -> String {
    let v = ParseFlow(flowText);
    serde_json::to_string_pretty(&v).unwrap_or_default()
}

pub fn ConvertJSONToFlow(jsonText: &str) -> String {
    let v: Value = serde_json::from_str(jsonText).unwrap_or(Value::Null);
    StringifyFlow(&v)
}

pub fn ParseFlowWithModel(text: &str, registry: Option<&ModelRegistry>) -> Value {
    // First, parse normally
    let data = ParseFlow(text);

    // Note: For simplicity in Rust, model extraction and application
    // would require additional helper functions. This basic implementation
    // returns the parsed data as-is. Full implementation would follow
    // the pattern from other languages but requires more Rust-specific code.

    // TODO: Complete implementation with model extraction and application
    // following the pattern from Python/TypeScript/C#/Go implementations

    data
}

pub fn LoadFlowWithModel(path: &str, registry: Option<&ModelRegistry>) -> Result<Value, std::io::Error> {
    let s = fs::read_to_string(path)?;
    Ok(ParseFlowWithModel(&s, registry))
}
