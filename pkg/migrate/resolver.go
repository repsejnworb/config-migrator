package migrate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ---- Path helpers (slash notation) ----

func split(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func isIndex(seg string) (int, bool) {
	i, err := strconv.Atoi(seg)
	if err != nil {
		return 0, false
	}
	return i, true
}

func hasWildcard(path string) bool {
	return strings.Contains(path, "*")
}

// getAtPath returns the value at a non-wildcard path.
func getAtPath(root map[string]interface{}, path string) (interface{}, bool, error) {
	cur := interface{}(root)
	for _, seg := range split(path) {
		switch node := cur.(type) {
		case map[string]interface{}:
			nxt, ok := node[seg]
			if !ok {
				return nil, false, nil
			}
			cur = nxt
		case []interface{}:
			if idx, ok := isIndex(seg); ok {
				if idx < 0 || idx >= len(node) {
					return nil, false, nil
				}
				cur = node[idx]
			} else if seg == "*" {
				return nil, false, fmt.Errorf("wildcard not allowed here: %s", path)
			} else {
				return nil, false, fmt.Errorf("array index expected at segment %q", seg)
			}
		default:
			return nil, false, nil
		}
	}
	return cur, true, nil
}

// setAtPath sets value at a non-wildcard path, creating missing objects for map segments.
func setAtPath(root map[string]interface{}, path string, val interface{}) error {
	cur := interface{}(root)
	segs := split(path)
	for i, seg := range segs {
		last := i == len(segs)-1
		switch node := cur.(type) {
		case map[string]interface{}:
			if last {
				node[seg] = val
				return nil
			}
			// ensure next container exists
			if _, ok := isIndex(segs[i+1]); ok {
				// next is an array index; we expect current[seg] to be []interface{}
				nxt, ok := node[seg]
				if !ok {
					return fmt.Errorf("cannot create array automatically for %s", path)
				}
				cur = nxt
			} else {
				nxt, ok := node[seg]
				if !ok {
					nxt = map[string]interface{}{}
					node[seg] = nxt
				}
				cur = nxt
			}
		case []interface{}:
			// setting directly into array by index is supported only if final
			idx, ok := isIndex(seg)
			if !ok {
				return fmt.Errorf("expected array index at %q", seg)
			}
			if idx < 0 || idx >= len(node) {
				return fmt.Errorf("index out of range at %q", seg)
			}
			if last {
				node[idx] = val
				return nil
			}
			cur = node[idx]
		default:
			return fmt.Errorf("cannot descend into %T at %q", node, seg)
		}
	}
	return nil
}

// deleteAtPath removes a map key at a non-wildcard path. Deleting array elements is not supported in this engine.
func deleteAtPath(root map[string]interface{}, path string) error {
	cur := interface{}(root)
	segs := split(path)
	for i, seg := range segs {
		last := i == len(segs)-1
		switch node := cur.(type) {
		case map[string]interface{}:
			if last {
				delete(node, seg)
				return nil
			}
			nxt, ok := node[seg]
			if !ok {
				return nil
			}
			cur = nxt
		case []interface{}:
			idx, ok := isIndex(seg)
			if !ok {
				return fmt.Errorf("expected index at %q; array deletion unsupported", seg)
			}
			if idx < 0 || idx >= len(node) {
				return nil
			}
			if last {
				return errors.New("array element deletion not supported in deleteAtPath")
			}
			cur = node[idx]
		default:
			return nil
		}
	}
	return nil
}

func findArrays(cfg map[string]interface{}, path string) ([][]interface{}, error) {
	var result [][]interface{}
	paths := resolveWildcardPaths(cfg, path)
	for _, p := range paths {
		v, ok, err := getAtPath(cfg, p)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		arr, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("mapArray: expected array at %s, got %T", p, v)
		}
		result = append(result, arr)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("mapArray: no arrays found at %s", path)
	}
	return result, nil
}

// resolveWildcardPaths returns all concrete paths matching wildcards
func resolveWildcardPaths(cfg map[string]interface{}, path string) []string {
	segs := split(path)
	var paths []string
	var walk func(curr map[string]interface{}, sofar []string)
	walk = func(curr map[string]interface{}, sofar []string) {
		if len(segs) == len(sofar) {
			paths = append(paths, strings.Join(sofar, "/"))
			return
		}
		seg := segs[len(sofar)]
		if seg == "*" {
			for k, v := range curr {
				switch vv := v.(type) {
				case map[string]interface{}:
					walk(vv, append(sofar, k))
				case []interface{}:
					if len(sofar)+1 == len(segs) {
						paths = append(paths, strings.Join(append(sofar, k), "/"))
					}
				}
			}
		} else {
			v, ok := curr[seg]
			if !ok {
				return
			}
			switch vv := v.(type) {
			case map[string]interface{}:
				walk(vv, append(sofar, seg))
			case []interface{}:
				if len(sofar)+1 == len(segs) {
					paths = append(paths, strings.Join(append(sofar, seg), "/"))
				}
			}
		}
	}
	walk(cfg, []string{})
	return paths
}
