package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Engine struct {
	migrations map[string]Migration // key: from->to
	graph      map[string][]string  // adjacency list
	validator  *Validator           // optional schema validator
}

func NewEngine() *Engine {
	return &Engine{migrations: make(map[string]Migration), graph: make(map[string][]string), validator: nil}
}

func (e *Engine) WithValidator(v *Validator) *Engine {
	e.validator = v
	return e
}

// LoadAll reads all *.json migrations in dir, registers them, and auto-generates reverse ones.
func (e *Engine) LoadAll(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, ent.Name()))
		if err != nil {
			return err
		}
		var m Migration
		if err := json.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("%s: %w", ent.Name(), err)
		}
		if err := e.addMigration(m); err != nil {
			return err
		}
		fmt.Println("Loaded migration:", m.From, "->", m.To)
		// auto-generate reverse if possible and not already present
		rev, err := GenerateReverse(m)
		if err == nil {
			_ = e.addMigrationIfMissing(rev)
		}
	}
	return nil
}

func (e *Engine) addMigrationIfMissing(m Migration) error {
	key := m.From + "->" + m.To
	if _, ok := e.migrations[key]; ok {
		return nil
	}
	return e.addMigration(m)
}

func (e *Engine) addMigration(m Migration) error {
	if m.From == "" || m.To == "" {
		return errors.New("migration missing from/to")
	}
	key := m.From + "->" + m.To
	e.migrations[key] = m
	e.graph[m.From] = append(e.graph[m.From], m.To)
	return nil
}

// Apply finds a chain from from->to and applies all migrations in order.
func (e *Engine) Apply(config map[string]interface{}, from, to string) (map[string]interface{}, error) {
	if from == to {
		return deepCopy(config), nil
	}
	chain, err := e.findChain(from, to)
	if err != nil {
		return nil, err
	}

	doc := deepCopy(config)
	for i := 0; i < len(chain)-1; i++ {
		a, b := chain[i], chain[i+1]
		mig, ok := e.migrations[a+"->"+b]
		if !ok {
			return nil, fmt.Errorf("missing migration %s->%s", a, b)
		}
		if err := e.applyMigration(doc, mig); err != nil {
			return nil, fmt.Errorf("apply %s->%s: %w", a, b, err)
		}
	}
	// validate final result against "to" schema if validator present
	if e.validator != nil {
		if err := e.validator.Validate(to, doc); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

func (e *Engine) findChain(from, to string) ([]string, error) {
	// BFS
	type node struct {
		v    string
		prev *node
	}
	q := []node{{v: from}}
	seen := map[string]bool{from: true}
	var end *node
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if cur.v == to {
			end = &cur
			break
		}
		for _, nxt := range e.graph[cur.v] {
			if !seen[nxt] {
				seen[nxt] = true
				q = append(q, node{v: nxt, prev: &cur})
			}
		}
	}
	if end == nil {
		return nil, fmt.Errorf("no migration path from %s to %s", from, to)
	}
	// reconstruct
	var rev []string
	for n := end; n != nil; n = n.prev {
		rev = append(rev, n.v)
	}
	// reverse
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev, nil
}

func (e *Engine) applyMigration(doc map[string]interface{}, m Migration) error {
	for i, step := range m.Steps {
		if err := e.applyStep(doc, step); err != nil {
			return fmt.Errorf("step %d (%s): %w", i, step.Op, err)
		}
	}
	return nil
}

func (e *Engine) applyStep(cfg map[string]interface{}, step MigrationStep) error {
	switch step.Op {
	case "move":
		if hasWildcard(step.From) || hasWildcard(step.To) {
			return fmt.Errorf("move does not support wildcards: from=%q to=%q", step.From, step.To)
		}
		v, ok, err := getAtPath(cfg, step.From)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("move: source not found %s", step.From)
		}
		if err := setAtPath(cfg, step.To, v); err != nil {
			return err
		}
		if err := deleteAtPath(cfg, step.From); err != nil {
			return err
		}
		return nil

	case "wrap":
		if hasWildcard(step.Path) {
			return fmt.Errorf("wrap: wildcards not allowed in path: %s", step.Path)
		}
		v, ok, err := getAtPath(cfg, step.Path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("wrap: path not found %s", step.Path)
		}
		obj := map[string]interface{}{step.WrapAs: v}
		return setAtPath(cfg, step.Path, obj)

	case "unwrap":
		if hasWildcard(step.Path) || hasWildcard(step.UnwrapTo) {
			return fmt.Errorf("unwrap: wildcards not allowed in paths")
		}
		v, ok, err := getAtPath(cfg, step.Path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("unwrap: source not found %s", step.Path)
		}
		return setAtPath(cfg, step.UnwrapTo, v)

	case "mapArray":
		arrays, err := findArrays(cfg, step.Path)
		if err != nil {
			return err
		}
		for _, arr := range arrays {
			for i := range arr {
				nv, err := applyItemRule(arr[i], step.Rule)
				if err != nil {
					return err
				}
				arr[i] = nv
			}
		}
		return nil

	case "set":
		if hasWildcard(step.Path) {
			return fmt.Errorf("set: wildcards not allowed: %s", step.Path)
		}
		return setAtPath(cfg, step.Path, step.Rule["value"]) // use Rule.value for literals

	case "delete":
		if hasWildcard(step.Path) {
			return fmt.Errorf("delete: wildcards not allowed: %s", step.Path)
		}
		return deleteAtPath(cfg, step.Path)
	}
	return fmt.Errorf("unsupported op %q", step.Op)
}

func applyItemRule(v interface{}, rule map[string]interface{}) (interface{}, error) {
	if rule == nil {
		return v, nil
	}
	if b, _ := rule["stringToObject"].(bool); b {
		sep, _ := ruleString(rule, "separator", ":")
		val, ok := rule["value"]
		if !ok {
			val = true
		}
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("stringToObject: expected string, got %T", v)
		}
		key := s
		if sep != "" {
			parts := strings.SplitN(s, sep, 2)
			key = parts[0]
		}
		return map[string]interface{}{key: val}, nil
	}
	if b, _ := rule["objectToString"].(bool); b {
		suf, _ := ruleString(rule, "suffix", "")
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("objectToString: expected object, got %T", v)
		}
		var key string
		for k, val := range m {
			switch x := val.(type) {
			case bool:
				if x {
					key = k
				}
			default:
				if val != nil {
					key = k
				}
			}
			if key != "" {
				break
			}
		}
		return key + suf, nil
	}
	return v, nil
}

func ruleString(m map[string]interface{}, key, def string) (string, bool) {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return def, false
}

func deepCopy(in map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(in)
	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)
	return out
}

// ---- Reverse generation ----

func GenerateReverse(m Migration) (Migration, error) {
	rev := Migration{From: m.To, To: m.From, Name: m.Name + "_reverse"}
	for i := len(m.Steps) - 1; i >= 0; i-- {
		s := m.Steps[i]
		if s.Reversible != nil && *s.Reversible == false {
			continue
		}
		rs, ok := invertStep(s)
		if !ok {
			continue
		}
		rev.Steps = append(rev.Steps, rs)
	}
	if len(rev.Steps) == 0 {
		return rev, fmt.Errorf("no reversible steps in %s->%s", m.From, m.To)
	}
	return rev, nil
}

func invertStep(s MigrationStep) (MigrationStep, bool) {
	switch s.Op {
	case "move":
		return MigrationStep{Op: "move", From: s.To, To: s.From}, true
	case "wrap":
		// wrap path=P wrapAs=K  ==>  unwrap path=P/K unwrapTo=P
		p := strings.TrimSuffix(s.Path, "/")
		return MigrationStep{Op: "unwrap", Path: p + "/" + s.WrapAs, UnwrapTo: p}, true
	case "unwrap":
		// unwrap path=P/K unwrapTo=P  ==>  wrap path=P wrapAs=K
		segs := split(s.Path)
		if len(segs) == 0 {
			return MigrationStep{}, false
		}
		k := segs[len(segs)-1]
		return MigrationStep{Op: "wrap", Path: s.UnwrapTo, WrapAs: k}, true
	case "mapArray":
		r := map[string]interface{}{}
		if b, _ := s.Rule["stringToObject"].(bool); b {
			r["objectToString"] = true
			if suf, ok := ruleString(s.Rule, "suffix", ""); ok {
				r["suffix"] = suf
			} else if sep, ok := ruleString(s.Rule, "separator", ""); ok {
				r["suffix"] = sep
			}
		} else if b, _ := s.Rule["objectToString"].(bool); b {
			r["stringToObject"] = true
			if sep, ok := ruleString(s.Rule, "separator", ""); ok {
				r["separator"] = sep
			} else if suf, ok := ruleString(s.Rule, "suffix", ""); ok {
				r["separator"] = suf
			}
			if v, ok := s.Rule["value"]; ok {
				r["value"] = v
			} else {
				r["value"] = true
			}
		} else {
			return MigrationStep{}, false
		}
		return MigrationStep{Op: "mapArray", Path: s.Path, Rule: r}, true
	default:
		return MigrationStep{}, false
	}
}
