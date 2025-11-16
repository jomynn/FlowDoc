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
