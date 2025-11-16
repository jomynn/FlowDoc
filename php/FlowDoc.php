<?php
/**
 * Simple FlowDoc PHP implementation (optional)
 * Requires ext-msgpack for binary functions.
 */

function tokenize_lines(string $text): array {
    $text = str_replace("\t", "  ", $text);
    $lines = preg_split('/\r?\n/', $text);
    $out = [];
    foreach ($lines as $line) {
        $no = explode('#', $line)[0];
        $no = rtrim($no);
        if (trim($no) !== '') $out[] = $no;
    }
    return $out;
}

function parse_value(string $raw) {
    $v = trim($raw);
    if ($v === 'true') return true;
    if ($v === 'false') return false;
    if (strlen($v) >= 2 && $v[0] === '"' && substr($v, -1) === '"') return substr($v, 1, -1);
    if (strlen($v) >= 2 && $v[0] === '[' && substr($v, -1) === ']') {
        $inner = trim(substr($v, 1, -1));
        if ($inner === '') return [];
        $parts = array_map('trim', explode(',', $inner));
        return array_map('parse_value', $parts);
    }
    if (is_numeric($v)) {
        if (strpos($v, '.') !== false) return (float)$v;
        return (int)$v;
    }
    return $v;
}

function ParseFlow(string $text): array {
    $lines = tokenize_lines($text);
    $root = [];
    $stack = [["indent"=>0, "node"=>&$root]];
    foreach ($lines as $line) {
        $leading = strlen($line) - strlen(ltrim($line));
        $indent = intval($leading / 2);
        $trimmed = trim($line);
        if (substr($trimmed, -1) === ':') {
            $key = trim(substr($trimmed, 0, -1));
            $obj = [];
            while (count($stack) && $stack[count($stack)-1]['indent'] >= $indent) array_pop($stack);
            $parent = &$stack[count($stack)-1]['node'];
            $parent[$key] = $obj;
            $stack[] = ["indent"=> $indent+1, "node"=> &$parent[$key]];
        } else {
            if (strpos($trimmed, '=') === false) continue;
            [$key, $raw] = array_map('trim', explode('=', $trimmed, 2));
            while (count($stack) && $stack[count($stack)-1]['indent'] > $indent) array_pop($stack);
            $parent = &$stack[count($stack)-1]['node'];
            $parent[$key] = parse_value($raw);
        }
    }
    return $root;
}

function StringifyFlow(array $obj): string {
    $out = '';
    $writeObj = function($o, $indent) use (&$out, &$writeObj) {
        $pad = str_repeat(' ', $indent);
        foreach ($o as $k => $v) {
            if (is_array($v) && array_values($v) !== $v) {
                // associative -> nested
                $out .= $pad . $k . ":\n";
                $writeObj($v, $indent+2);
            } elseif (is_array($v)) {
                $parts = array_map(function($e){
                    if (is_string($e)) return strpos($e, ' ') !== false ? '"'.$e.'"' : $e;
                    return (string)$e;
                }, $v);
                $out .= $pad . $k . ' = [' . implode(', ', $parts) . "]\n";
            } elseif (is_string($v)) {
                $out .= $pad . $k . ' = ' . (strpos($v, ' ') !== false ? '"'.$v.'"' : $v) . "\n";
            } else {
                $out .= $pad . $k . ' = ' . var_export($v, true) . "\n";
            }
        }
    };
    $writeObj($obj, 0);
    return $out;
}

function LoadFlow(string $path): array { return ParseFlow(file_get_contents($path)); }
function SaveFlow(string $path, array $obj) { file_put_contents($path, StringifyFlow($obj)); }

function LoadFlowb(string $path) { $data = file_get_contents($path); return msgpack_unpack($data); }
function SaveFlowb(string $path, $obj) { $data = msgpack_pack($obj); file_put_contents($path, $data); }

function ConvertFlowToJSON(string $flowText): string { return json_encode(ParseFlow($flowText), JSON_PRETTY_PRINT); }
function ConvertJSONToFlow(string $jsonText): string { return StringifyFlow(json_decode($jsonText, true)); }

?>
