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
		// version without extension, e.g. "v1" from "v1.json"
		version := ent.Name()
		version = version[:len(version)-len(filepath.Ext(version))]

		path := filepath.Join(dir, ent.Name())

		// Open file and register it under the resource name equal to the file name (e.g. "v1.json")
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open schema %s: %w", path, err)
		}

		compiler := jsonschema.NewCompiler()

		resourceName := ent.Name() // use the filename (with .json) as the resource key
		if err := compiler.AddResource(resourceName, f); err != nil {
			f.Close()
			return fmt.Errorf("failed to add schema resource %s: %w", path, err)
		}

		// Compile using the same resource name
		sch, err := compiler.Compile(resourceName)
		// Close file after compile (compiler will have read it)
		_ = f.Close()
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
