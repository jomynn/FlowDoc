using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.Json;
using MessagePack;
using MessagePack.Resolvers;

namespace FlowDoc
{
    // ============================================
    // Mapping Model Support
    // ============================================

    public class FieldDefinition
    {
        public string FullName { get; set; }
        public string Alias { get; set; }
        public string FieldType { get; set; }
        public int? FieldId { get; set; }

        public FieldDefinition(string fullName, string alias, string fieldType = "string", int? fieldId = null)
        {
            FullName = fullName;
            Alias = alias;
            FieldType = fieldType;
            FieldId = fieldId;
        }
    }

    public class ModelDefinition
    {
        public string Name { get; set; }
        public Dictionary<string, FieldDefinition> Fields { get; set; }  // indexed by full name
        public Dictionary<string, string> AliasMap { get; set; }  // alias -> full name

        public ModelDefinition(string name)
        {
            Name = name;
            Fields = new Dictionary<string, FieldDefinition>();
            AliasMap = new Dictionary<string, string>();
        }

        public void AddField(FieldDefinition field)
        {
            Fields[field.FullName] = field;
            AliasMap[field.Alias] = field.FullName;
        }
    }

    public class ModelRegistry
    {
        private Dictionary<string, ModelDefinition> models = new Dictionary<string, ModelDefinition>();

        public void RegisterModel(ModelDefinition model)
        {
            models[model.Name] = model;
        }

        public ModelDefinition? GetModel(string name)
        {
            return models.TryGetValue(name, out var model) ? model : null;
        }
    }

    public static class FlowDoc
    {
        static FlowDoc()
        {
            // Use contractless resolver so dictionaries serialize without attributes
            var opts = MessagePackSerializerOptions.Standard.WithResolver(ContractlessStandardResolver.Instance);
        }

        static object ParseTypedValue(string raw, string fieldType)
        {
            var v = raw.Trim();

            if (fieldType == "bool")
            {
                if (v == "true") return true;
                if (v == "false") return false;
                throw new ArgumentException($"Invalid boolean value: {v}");
            }

            if (fieldType == "int")
            {
                return int.Parse(v);
            }

            if (fieldType == "float")
            {
                return double.Parse(v);
            }

            if (fieldType == "date")
            {
                // Validate YYYY-MM-DD format
                if (!System.Text.RegularExpressions.Regex.IsMatch(v, @"^\d{4}-\d{2}-\d{2}$"))
                    throw new ArgumentException($"Invalid date format (expected YYYY-MM-DD): {v}");
                return v;  // Keep as string for JSON compatibility
            }

            if (fieldType == "datetime")
            {
                // Validate ISO 8601 datetime
                if (!System.Text.RegularExpressions.Regex.IsMatch(v, @"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}"))
                    throw new ArgumentException($"Invalid datetime format (expected ISO 8601): {v}");
                return v;  // Keep as string for JSON compatibility
            }

            // string type - remove quotes if present
            if (v.StartsWith("\"") && v.EndsWith("\""))
                return v.Substring(1, v.Length - 2);

            return v;
        }

        static ModelRegistry? ExtractModelsFromDict(Dictionary<string, object> data)
        {
            if (!data.ContainsKey("$models"))
                return null;

            var registry = new ModelRegistry();
            if (!(data["$models"] is Dictionary<string, object> modelsData))
                return null;

            foreach (var modelEntry in modelsData)
            {
                var modelName = modelEntry.Key;
                if (!(modelEntry.Value is Dictionary<string, object> modelSpec))
                    continue;

                if (!modelSpec.ContainsKey("fields"))
                    continue;

                var modelDef = new ModelDefinition(modelName);

                if (!(modelSpec["fields"] is Dictionary<string, object> fieldsData))
                    continue;

                foreach (var fieldEntry in fieldsData)
                {
                    var fullName = fieldEntry.Key;
                    if (!(fieldEntry.Value is Dictionary<string, object> fieldSpec))
                        continue;

                    var alias = fieldSpec.TryGetValue("alias", out var a) ? a.ToString() : fullName;
                    var fieldType = fieldSpec.TryGetValue("type", out var t) ? t.ToString() : "string";
                    int? fieldId = fieldSpec.TryGetValue("id", out var id) && id is int idVal ? idVal : null;

                    var fieldDef = new FieldDefinition(fullName, alias ?? fullName, fieldType ?? "string", fieldId);
                    modelDef.AddField(fieldDef);
                }

                registry.RegisterModel(modelDef);
            }

            return registry;
        }

        static Dictionary<string, object> ApplyModelToDict(Dictionary<string, object> data, ModelDefinition model)
        {
            var result = new Dictionary<string, object>();

            foreach (var entry in data)
            {
                var key = entry.Key;
                var value = entry.Value;

                // Check if this key is an alias
                var fullName = model.AliasMap.TryGetValue(key, out var fn) ? fn : key;
                model.Fields.TryGetValue(fullName, out var fieldDef);

                // Process value
                if (value is Dictionary<string, object> nestedDict)
                {
                    result[fullName] = ApplyModelToDict(nestedDict, model);
                }
                else if (value is List<object> list)
                {
                    var newList = new List<object>();
                    foreach (var item in list)
                    {
                        if (item is Dictionary<string, object> itemDict)
                            newList.Add(ApplyModelToDict(itemDict, model));
                        else
                            newList.Add(item);
                    }
                    result[fullName] = newList;
                }
                else
                {
                    // Apply type conversion if field definition exists
                    if (fieldDef != null)
                    {
                        try
                        {
                            result[fullName] = ParseTypedValue(value.ToString() ?? "", fieldDef.FieldType);
                        }
                        catch
                        {
                            result[fullName] = value;
                        }
                    }
                    else
                    {
                        result[fullName] = value;
                    }
                }
            }

            return result;
        }

        static List<string> TokenizeLines(string text)
        {
            text = text.Replace("\t", "  ");
            var outLines = new List<string>();
            using (var reader = new StringReader(text))
            {
                string? line;
                while ((line = reader.ReadLine()) != null)
                {
                    var noComment = line.Split('#')[0];
                    if (!string.IsNullOrWhiteSpace(noComment)) outLines.Add(noComment.TrimEnd());
                }
            }
            return outLines;
        }

        static object ParseValue(string raw)
        {
            var v = raw.Trim();
            if (v == "true") return true;
            if (v == "false") return false;
            if (v.StartsWith("\"") && v.EndsWith("\"")) return v.Substring(1, v.Length - 2);
            if (v.StartsWith("[") && v.EndsWith("]"))
            {
                var inner = v.Substring(1, v.Length - 2).Trim();
                if (string.IsNullOrEmpty(inner)) return new List<object>();
                var parts = inner.Split(',');
                var list = new List<object>();
                foreach (var p in parts) list.Add(ParseValue(p));
                return list;
            }
            if (int.TryParse(v, out var i)) return i;
            if (double.TryParse(v, out var f)) return f;
            return v;
        }

        public static Dictionary<string, object> ParseFlow(string text)
        {
            var lines = TokenizeLines(text);
            var root = new Dictionary<string, object>();
            var stack = new List<(int indent, Dictionary<string, object> node)> { (0, root) };

            foreach (var line in lines)
            {
                var leading = line.Length - line.TrimStart().Length;
                var indent = leading / 2;
                var trimmed = line.Trim();
                if (trimmed.EndsWith(":"))
                {
                    var key = trimmed.Substring(0, trimmed.Length - 1).Trim();
                    var obj = new Dictionary<string, object>();
                    while (stack.Count > 0 && stack[stack.Count - 1].indent >= indent) stack.RemoveAt(stack.Count - 1);
                    stack[stack.Count - 1].node[key] = obj;
                    stack.Add((indent + 1, obj));
                }
                else
                {
                    var parts = trimmed.Split('=', 2);
                    if (parts.Length < 2) continue;
                    var key = parts[0].Trim();
                    var raw = parts[1].Trim();
                    while (stack.Count > 0 && stack[stack.Count - 1].indent > indent) stack.RemoveAt(stack.Count - 1);
                    stack[stack.Count - 1].node[key] = ParseValue(raw);
                }
            }

            return root;
        }

        public static string StringifyFlow(Dictionary<string, object> obj)
        {
            string StringifyObj(Dictionary<string, object> o, int indent)
            {
                var pad = new string(' ', indent);
                var lines = new List<string>();
                foreach (var kv in o)
                {
                    if (kv.Value is Dictionary<string, object> nested)
                    {
                        lines.Add(pad + kv.Key + ":");
                        lines.Add(StringifyObj(nested, indent + 2));
                    }
                    else if (kv.Value is List<object> arr)
                    {
                        var parts = new List<string>();
                        foreach (var e in arr)
                        {
                            if (e is string s) parts.Add(s.Contains(" ") ? $"\"{s}\"" : s);
                            else parts.Add(e.ToString());
                        }
                        lines.Add(pad + kv.Key + " = [" + string.Join(", ", parts) + "]");
                    }
                    else if (kv.Value is string s)
                    {
                        lines.Add(pad + kv.Key + " = " + (s.Contains(" ") ? $"\"{s}\"" : s));
                    }
                    else
                    {
                        lines.Add(pad + kv.Key + " = " + kv.Value?.ToString());
                    }
                }
                return string.Join("\n", lines);
            }
            return StringifyObj(obj, 0) + "\n";
        }

        public static Dictionary<string, object> LoadFlow(string path)
        {
            var txt = File.ReadAllText(path);
            return ParseFlow(txt);
        }

        public static void SaveFlow(string path, Dictionary<string, object> obj)
        {
            File.WriteAllText(path, StringifyFlow(obj));
        }

        public static object LoadFlowb(string path)
        {
            var data = File.ReadAllBytes(path);
            var opts = MessagePackSerializerOptions.Standard.WithResolver(ContractlessStandardResolver.Instance);
            return MessagePackSerializer.Deserialize<object>(data, opts);
        }

        public static void SaveFlowb(string path, object obj)
        {
            var opts = MessagePackSerializerOptions.Standard.WithResolver(ContractlessStandardResolver.Instance);
            var data = MessagePackSerializer.Serialize(obj, opts);
            File.WriteAllBytes(path, data);
        }

        public static string ConvertFlowToJSON(string flowText)
        {
            var obj = ParseFlow(flowText);
            return JsonSerializer.Serialize(obj, new JsonSerializerOptions { WriteIndented = true });
        }

        public static string ConvertJSONToFlow(string jsonText)
        {
            var obj = JsonSerializer.Deserialize<Dictionary<string, object>>(jsonText);
            return StringifyFlow(obj ?? new Dictionary<string, object>());
        }

        public static Dictionary<string, object> ParseFlowWithModel(string text, ModelRegistry? registry = null)
        {
            // First, parse normally
            var data = ParseFlow(text);

            // Extract or use provided registry
            var modelRegistry = registry ?? ExtractModelsFromDict(data);

            // Check for use_model directive
            if (modelRegistry != null && data.ContainsKey("use_model"))
            {
                var modelName = data["use_model"].ToString();
                var model = modelRegistry.GetModel(modelName ?? "");

                if (model == null)
                    throw new ArgumentException($"Model '{modelName}' not found in registry");

                // Remove use_model and $models from result
                var result = new Dictionary<string, object>();
                foreach (var entry in data)
                {
                    if (entry.Key != "use_model" && entry.Key != "$models")
                        result[entry.Key] = entry.Value;
                }

                // Apply model to all top-level values
                var finalResult = new Dictionary<string, object>();
                foreach (var entry in result)
                {
                    var value = entry.Value;
                    if (value is Dictionary<string, object> dict)
                    {
                        finalResult[entry.Key] = ApplyModelToDict(dict, model);
                    }
                    else if (value is List<object> list)
                    {
                        var newList = new List<object>();
                        foreach (var item in list)
                        {
                            if (item is Dictionary<string, object> itemDict)
                                newList.Add(ApplyModelToDict(itemDict, model));
                            else
                                newList.Add(item);
                        }
                        finalResult[entry.Key] = newList;
                    }
                    else
                    {
                        finalResult[entry.Key] = value;
                    }
                }

                return finalResult;
            }

            // No model application needed
            // Remove $models from output if present
            if (data.ContainsKey("$models"))
            {
                var result = new Dictionary<string, object>();
                foreach (var entry in data)
                {
                    if (entry.Key != "$models")
                        result[entry.Key] = entry.Value;
                }
                return result;
            }

            return data;
        }

        public static Dictionary<string, object> LoadFlowWithModel(string path, ModelRegistry? registry = null)
        {
            var txt = File.ReadAllText(path);
            return ParseFlowWithModel(txt, registry);
        }
    }
}
