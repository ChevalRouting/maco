package mcpserver

import (
	"fmt"
	"strings"
)

func expandSchema(v any, definitions map[string]any, depth int) (any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("recursive API schema exceeds maximum depth")
	}

	switch x := v.(type) {
	case map[string]any:
		if ref, ok := x["$ref"].(string); ok {
			key := strings.TrimPrefix(ref, "#/components/schemas/")
			target, ok := definitions[key]
			if !ok {
				return nil, fmt.Errorf("unknown schema reference %q", ref)
			}

			expanded, err := expandSchema(target, definitions, depth+1)
			if err != nil {
				return nil, err
			}

			schema, ok := expanded.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("schema reference must resolve to an object")
			}

			for key, value := range x {
				if key == "$ref" {
					continue
				}

				sibling, err := expandSchema(value, definitions, depth+1)
				if err != nil {
					return nil, err
				}

				schema[key] = sibling
			}

			return schema, nil
		}

		if choices, ok := x["oneOf"].([]any); ok && len(choices) == 2 {
			first, ok := choices[0].(map[string]any)
			second, ok2 := choices[1].(map[string]any)
			if ok && ok2 && len(first) == 1 && first["type"] == "object" && second["$ref"] != nil {
				return expandSchema(second, definitions, depth+1)
			}
		}

		out := map[string]any{}
		for k, item := range x {
			expanded, err := expandSchema(item, definitions, depth+1)
			if err != nil {
				return nil, err
			}

			out[k] = expanded
		}

		return out, nil
	case []any:
		out := make([]any, len(x))
		for j, item := range x {
			expanded, err := expandSchema(item, definitions, depth+1)
			if err != nil {
				return nil, err
			}

			out[j] = expanded
		}

		return out, nil
	default:
		return v, nil
	}
}
