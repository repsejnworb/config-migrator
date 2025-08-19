package migrator

import (
	"fmt"
	"strings"
)

// resolvePath navigates through arbitrary nested maps/arrays using slash notation.
// Supports wildcards: * for array iteration.
func resolvePath(data interface{}, path string) ([]interface{}, error) {
	parts := strings.Split(path, "/")
	return walk(data, parts)
}

func walk(node interface{}, parts []string) ([]interface{}, error) {
	if len(parts) == 0 {
		return []interface{}{node}, nil
	}

	head, tail := parts[0], parts[1:]

	switch n := node.(type) {
	case map[string]interface{}:
		child, ok := n[head]
		if !ok {
			return nil, fmt.Errorf("key not found: %s", head)
		}
		return walk(child, tail)

	case []interface{}:
		if head == "*" {
			var results []interface{}
			for _, elem := range n {
				res, err := walk(elem, tail)
				if err != nil {
					return nil, err
				}
				results = append(results, res...)
			}
			return results, nil
		}
		return nil, fmt.Errorf("invalid path segment '%s' for array", head)
	}

	return nil, fmt.Errorf("unexpected type in path resolution")
}
