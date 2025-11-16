using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using MessagePack;
using MessagePack.Resolvers;

namespace FlowDoc
{
    public static class FlowDoc
    {
        static FlowDoc()
        {
            // Use contractless resolver so dictionaries serialize without attributes
            var opts = MessagePackSerializerOptions.Standard.WithResolver(ContractlessStandardResolver.Instance);
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
    }
}
