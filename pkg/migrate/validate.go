package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validator holds compiled schemas keyed by version (like "v1", "v2")
type Validator struct {
	schemas map[string]*jsonschema.Schema
}

func NewValidator() *Validator {
	return &Validator{schemas: make(map[string]*jsonschema.Schema)}
}

// LoadAll compiles all *.json schemas in dir.
func (v *Validator) LoadAll(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() || filepath.Ext(ent.Name()) != ".json" {
			continue
		}
		version := ent.Name()
		version = version[:len(version)-len(filepath.Ext(version))] // strip .json

		path := filepath.Join(dir, ent.Name())
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource(version, os.Open(path)); err != nil {
			return fmt.Errorf("load schema %s: %w", ent.Name(), err)
		}
		sch, err := compiler.Compile(version)
		if err != nil {
			return fmt.Errorf("compile schema %s: %w", ent.Name(), err)
		}
		v.schemas[version] = sch
	}
	return nil
}

// Validate ensures doc conforms to schema for given version.
func (v *Validator) Validate(version string, doc map[string]interface{}) error {
	sch, ok := v.schemas[version]
	if !ok {
		return fmt.Errorf("no schema for version %s", version)
	}
	// re-marshal to ensure canonical interface{} decoding
	b, _ := json.Marshal(doc)
	var redecoded interface{}
	if err := json.Unmarshal(b, &redecoded); err != nil {
		return err
	}
	if err := sch.Validate(redecoded); err != nil {
		return fmt.Errorf("schema validation failed for version %s: %w", version, err)
	}
	return nil
}
