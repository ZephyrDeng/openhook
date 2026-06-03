package render

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var placeholder = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.$\[\]-]+)\s*\}\}`)

type Context struct {
	Data   map[string]any `json:"data"`
	Global map[string]any `json:"global"`
}

func Template(content string, data map[string]any, global map[string]any) (any, error) {
	rendered := placeholder.ReplaceAllStringFunc(content, func(token string) string {
		match := placeholder.FindStringSubmatch(token)
		if len(match) != 2 {
			return token
		}
		value, ok := Resolve(map[string]any{"data": data, "global": global}, match[1])
		if !ok || value == nil {
			return ""
		}
		switch typed := value.(type) {
		case string:
			return typed
		case json.Number:
			return typed.String()
		default:
			raw, err := json.Marshal(typed)
			if err != nil {
				return fmt.Sprint(typed)
			}
			return string(raw)
		}
	})

	trimmed := strings.TrimSpace(rendered)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var value any
		if err := json.Unmarshal([]byte(trimmed), &value); err == nil {
			return value, nil
		}
	}
	return rendered, nil
}

func DecodeObject(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var data map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]any{}
	}
	return data, nil
}

func Resolve(root map[string]any, path string) (any, bool) {
	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")
	var current any = root
	for _, part := range parts {
		if part == "" {
			continue
		}
		key, indexes := splitIndexes(part)
		if key != "" {
			object, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}
			current, ok = object[key]
			if !ok {
				return nil, false
			}
		}
		for _, index := range indexes {
			list, ok := current.([]any)
			if !ok || index < 0 || index >= len(list) {
				return nil, false
			}
			current = list[index]
		}
	}
	return current, true
}

func splitIndexes(part string) (string, []int) {
	start := strings.Index(part, "[")
	if start < 0 {
		return part, nil
	}
	key := part[:start]
	var indexes []int
	rest := part[start:]
	for strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]")
		if end < 0 {
			break
		}
		index, err := strconv.Atoi(rest[1:end])
		if err == nil {
			indexes = append(indexes, index)
		}
		rest = rest[end+1:]
	}
	return key, indexes
}
