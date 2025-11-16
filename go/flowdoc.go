package flowdoc

import (
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "regexp"
    "strconv"
    "strings"

    msgpack "github.com/vmihailenco/msgpack/v5"
)

// ============================================
// Mapping Model Support
// ============================================

type FieldDefinition struct {
    FullName  string
    Alias     string
    FieldType string
    FieldID   *int
}

type ModelDefinition struct {
    Name     string
    Fields   map[string]*FieldDefinition // indexed by full name
    AliasMap map[string]string           // alias -> full name
}

func NewModelDefinition(name string) *ModelDefinition {
    return &ModelDefinition{
        Name:     name,
        Fields:   make(map[string]*FieldDefinition),
        AliasMap: make(map[string]string),
    }
}

func (m *ModelDefinition) AddField(field *FieldDefinition) {
    m.Fields[field.FullName] = field
    m.AliasMap[field.Alias] = field.FullName
}

type ModelRegistry struct {
    models map[string]*ModelDefinition
}

func NewModelRegistry() *ModelRegistry {
    return &ModelRegistry{
        models: make(map[string]*ModelDefinition),
    }
}

func (r *ModelRegistry) RegisterModel(model *ModelDefinition) {
    r.models[model.Name] = model
}

func (r *ModelRegistry) GetModel(name string) *ModelDefinition {
    return r.models[name]
}

func parseTypedValue(raw string, fieldType string) (interface{}, error) {
    v := strings.TrimSpace(raw)

    switch fieldType {
    case "bool":
        if v == "true" {
            return true, nil
        }
        if v == "false" {
            return false, nil
        }
        return nil, fmt.Errorf("invalid boolean value: %s", v)

    case "int":
        i, err := strconv.Atoi(v)
        if err != nil {
            return nil, fmt.Errorf("invalid integer value: %s", v)
        }
        return i, nil

    case "float":
        f, err := strconv.ParseFloat(v, 64)
        if err != nil {
            return nil, fmt.Errorf("invalid float value: %s", v)
        }
        return f, nil

    case "date":
        // Validate YYYY-MM-DD format
        matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, v)
        if !matched {
            return nil, fmt.Errorf("invalid date format (expected YYYY-MM-DD): %s", v)
        }
        return v, nil // Keep as string for JSON compatibility

    case "datetime":
        // Validate ISO 8601 datetime
        matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`, v)
        if !matched {
            return nil, fmt.Errorf("invalid datetime format (expected ISO 8601): %s", v)
        }
        return v, nil // Keep as string for JSON compatibility

    default: // "string" or unknown type
        // Remove quotes if present
        if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
            return v[1 : len(v)-1], nil
        }
        return v, nil
    }
}

func extractModelsFromMap(data map[string]interface{}) *ModelRegistry {
    modelsVal, ok := data["$models"]
    if !ok {
        return nil
    }

    modelsData, ok := modelsVal.(map[string]interface{})
    if !ok {
        return nil
    }

    registry := NewModelRegistry()

    for modelName, modelSpecVal := range modelsData {
        modelSpec, ok := modelSpecVal.(map[string]interface{})
        if !ok {
            continue
        }

        fieldsVal, ok := modelSpec["fields"]
        if !ok {
            continue
        }

        fieldsData, ok := fieldsVal.(map[string]interface{})
        if !ok {
            continue
        }

        modelDef := NewModelDefinition(modelName)

        for fullName, fieldSpecVal := range fieldsData {
            fieldSpec, ok := fieldSpecVal.(map[string]interface{})
            if !ok {
                continue
            }

            alias := fullName
            if a, ok := fieldSpec["alias"]; ok {
                if aliasStr, ok := a.(string); ok {
                    alias = aliasStr
                }
            }

            fieldType := "string"
            if t, ok := fieldSpec["type"]; ok {
                if typeStr, ok := t.(string); ok {
                    fieldType = typeStr
                }
            }

            var fieldID *int
            if id, ok := fieldSpec["id"]; ok {
                if idInt, ok := id.(int); ok {
                    fieldID = &idInt
                }
            }

            fieldDef := &FieldDefinition{
                FullName:  fullName,
                Alias:     alias,
                FieldType: fieldType,
                FieldID:   fieldID,
            }

            modelDef.AddField(fieldDef)
        }

        registry.RegisterModel(modelDef)
    }

    return registry
}

func applyModelToMap(data map[string]interface{}, model *ModelDefinition) map[string]interface{} {
    result := make(map[string]interface{})

    for key, value := range data {
        // Check if this key is an alias
        fullName := key
        if fn, ok := model.AliasMap[key]; ok {
            fullName = fn
        }

        fieldDef := model.Fields[fullName]

        // Process value
        if nestedMap, ok := value.(map[string]interface{}); ok {
            result[fullName] = applyModelToMap(nestedMap, model)
        } else if arr, ok := value.([]interface{}); ok {
            newArr := make([]interface{}, len(arr))
            for i, item := range arr {
                if itemMap, ok := item.(map[string]interface{}); ok {
                    newArr[i] = applyModelToMap(itemMap, model)
                } else {
                    newArr[i] = item
                }
            }
            result[fullName] = newArr
        } else {
            // Apply type conversion if field definition exists
            if fieldDef != nil {
                if typedVal, err := parseTypedValue(fmt.Sprintf("%v", value), fieldDef.FieldType); err == nil {
                    result[fullName] = typedVal
                } else {
                    result[fullName] = value
                }
            } else {
                result[fullName] = value
            }
        }
    }

    return result
}

// ============================================
// Core Parsing Functions
// ============================================

func tokenizeLines(text string) []string {
    text = strings.ReplaceAll(text, "\t", "  ")
    var out []string
    scanner := bufio.NewScanner(strings.NewReader(text))
    for scanner.Scan() {
        line := scanner.Text()
        parts := strings.SplitN(line, "#", 2)
        no := strings.TrimRight(parts[0], " \t")
        if strings.TrimSpace(no) != "" {
            out = append(out, no)
        }
    }
    return out
}

func parseValue(raw string) interface{} {
    v := strings.TrimSpace(raw)
    if v == "true" {
        return true
    }
    if v == "false" {
        return false
    }
    if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
        return v[1:len(v)-1]
    }
    if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
        inner := strings.TrimSpace(v[1 : len(v)-1])
        if inner == "" {
            return []interface{}{}
        }
        parts := strings.Split(inner, ",")
        arr := make([]interface{}, 0, len(parts))
        for _, p := range parts {
            arr = append(arr, parseValue(p))
        }
        return arr
    }
    // number?
    if matched, _ := regexp.MatchString(`^-?\d+(\.\d+)?$`, v); matched {
        if strings.Contains(v, ".") {
            f, _ := strconv.ParseFloat(v, 64)
            return f
        }
        i, _ := strconv.Atoi(v)
        return i
    }
    return v
}

func ParseFlow(text string) map[string]interface{} {
    lines := tokenizeLines(text)
    root := make(map[string]interface{})
    stack := []struct{
        indent int
        node map[string]interface{}
    }{{0, root}}

    for _, line := range lines {
        leading := len(regexp.MustCompile(`^\s*`).FindString(line))
        indent := leading / 2
        trimmed := strings.TrimSpace(line)
        if strings.HasSuffix(trimmed, ":") {
            key := strings.TrimSpace(trimmed[:len(trimmed)-1])
            obj := make(map[string]interface{})
            for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
                stack = stack[:len(stack)-1]
            }
            stack[len(stack)-1].node[key] = obj
            stack = append(stack, struct{
                indent int
                node map[string]interface{}
            }{indent+1, obj})
        } else {
            if !strings.Contains(trimmed, "=") { continue }
            parts := strings.SplitN(trimmed, "=", 2)
            key := strings.TrimSpace(parts[0])
            raw := strings.TrimSpace(parts[1])
            for len(stack) > 0 && stack[len(stack)-1].indent > indent {
                stack = stack[:len(stack)-1]
            }
            stack[len(stack)-1].node[key] = parseValue(raw)
        }
    }
    return root
}

func StringifyFlow(obj map[string]interface{}) string {
    var b bytes.Buffer
    var writeObj func(map[string]interface{}, int)
    writeObj = func(o map[string]interface{}, indent int) {
        pad := strings.Repeat(" ", indent)
        for k, v := range o {
            switch vv := v.(type) {
            case map[string]interface{}:
                b.WriteString(pad + k + ":\n")
                writeObj(vv, indent+2)
            case []interface{}:
                arr := make([]string, 0, len(vv))
                for _, e := range vv {
                    arr = append(arr, stringifyBasic(e))
                }
                b.WriteString(pad + k + " = [" + strings.Join(arr, ", ") + "]\n")
            default:
                b.WriteString(pad + k + " = " + stringifyBasic(v) + "\n")
            }
        }
    }
    writeObj(obj, 0)
    return b.String()
}

func stringifyBasic(v interface{}) string {
    switch x := v.(type) {
    case string:
        if strings.ContainsAny(x, " \t") || x == "" {
            return "\"" + x + "\""
        }
        return x
    case bool:
        if x { return "true" }
        return "false"
    case int, int64, float64, float32:
        return fmt.Sprintf("%v", x)
    default:
        return ""
    }
}

func LoadFlow(path string) (map[string]interface{}, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil { return nil, err }
    return ParseFlow(string(data)), nil
}

func SaveFlow(path string, obj map[string]interface{}) error {
    return ioutil.WriteFile(path, []byte(StringifyFlow(obj)), 0644)
}

func LoadFlowb(path string) (map[string]interface{}, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil { return nil, err }
    var out map[string]interface{}
    err = msgpack.Unmarshal(data, &out)
    return out, err
}

func SaveFlowb(path string, obj map[string]interface{}) error {
    data, err := msgpack.Marshal(obj)
    if err != nil { return err }
    return ioutil.WriteFile(path, data, 0644)
}

func ConvertFlowToJSON(flowText string) (string, error) {
    obj := ParseFlow(flowText)
    b, err := json.MarshalIndent(obj, "", "  ")
    return string(b), err
}

func ConvertJSONToFlow(jsonText string) (string, error) {
    var obj map[string]interface{}
    if err := json.Unmarshal([]byte(jsonText), &obj); err != nil { return "", err }
    return StringifyFlow(obj), nil
}

func ParseFlowWithModel(text string, registry *ModelRegistry) (map[string]interface{}, error) {
    // First, parse normally
    data := ParseFlow(text)

    // Extract or use provided registry
    modelRegistry := registry
    if modelRegistry == nil {
        modelRegistry = extractModelsFromMap(data)
    }

    // Check for use_model directive
    if modelRegistry != nil {
        if modelNameVal, ok := data["use_model"]; ok {
            modelName, ok := modelNameVal.(string)
            if !ok {
                return nil, fmt.Errorf("use_model must be a string")
            }

            model := modelRegistry.GetModel(modelName)
            if model == nil {
                return nil, fmt.Errorf("model '%s' not found in registry", modelName)
            }

            // Remove use_model and $models from result
            result := make(map[string]interface{})
            for key, value := range data {
                if key != "use_model" && key != "$models" {
                    result[key] = value
                }
            }

            // Apply model to all top-level values
            finalResult := make(map[string]interface{})
            for key, value := range result {
                if valueMap, ok := value.(map[string]interface{}); ok {
                    finalResult[key] = applyModelToMap(valueMap, model)
                } else if arr, ok := value.([]interface{}); ok {
                    newArr := make([]interface{}, len(arr))
                    for i, item := range arr {
                        if itemMap, ok := item.(map[string]interface{}); ok {
                            newArr[i] = applyModelToMap(itemMap, model)
                        } else {
                            newArr[i] = item
                        }
                    }
                    finalResult[key] = newArr
                } else {
                    finalResult[key] = value
                }
            }

            return finalResult, nil
        }
    }

    // No model application needed
    // Remove $models from output if present
    if _, ok := data["$models"]; ok {
        result := make(map[string]interface{})
        for key, value := range data {
            if key != "$models" {
                result[key] = value
            }
        }
        return result, nil
    }

    return data, nil
}

func LoadFlowWithModel(path string, registry *ModelRegistry) (map[string]interface{}, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    return ParseFlowWithModel(string(data), registry)
}
