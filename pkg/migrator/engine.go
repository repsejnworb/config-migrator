package migrator

import (
	"encoding/json"
	"fmt"
	"os"
)

type Engine struct {
	migrations map[string]Migration
}

func NewEngine() *Engine {
	return &Engine{migrations: make(map[string]Migration)}
}

func (e *Engine) LoadMigration(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var m Migration
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	key := fmt.Sprintf("%s->%s", m.From, m.To)
	e.migrations[key] = m
	return nil
}

func (e *Engine) Apply(config map[string]interface{}, from, to string) (map[string]interface{}, error) {
	key := fmt.Sprintf("%s->%s", from, to)
	mig, ok := e.migrations[key]
	if !ok {
		return nil, fmt.Errorf("migration %s not found", key)
	}

	// Copy config
	newCfg := deepCopy(config)

	for _, step := range mig.Steps {
		if err := e.applyStep(newCfg, step); err != nil {
			return nil, err
		}
	}
	return newCfg, nil
}

func (e *Engine) applyStep(cfg map[string]interface{}, step MigrationStep) error {
	switch step.Op {
	case "move":
	// TODO: implement move with resolver
	case "wrap":
	// TODO: implement wrap
	case "unwrap":
	// TODO: implement unwrap
	case "mapArray":
	// TODO: implement array element transformation
	default:
		return fmt.Errorf("unsupported op: %s", step.Op)
	}
	return nil
}

func deepCopy(in map[string]interface{}) map[string]interface{} {
	b, _ := json.Marshal(in)
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	return out
}
