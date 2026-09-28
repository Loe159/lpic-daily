package schemavalidation

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaBaseURL = "https://lpic-daily.dev/schemas/"

type Validator struct {
	fsys     fs.FS
	compiler *jsonschema.Compiler
	compiled map[string]*jsonschema.Schema
}

func New(fsys fs.FS) (*Validator, error) {
	if fsys == nil {
		return nil, fmt.Errorf("schema filesystem is required")
	}

	compiler := jsonschema.NewCompiler()
	schemaPaths, err := fs.Glob(fsys, "schemas/*.schema.json")
	if err != nil {
		return nil, fmt.Errorf("glob JSON schemas: %w", err)
	}
	if len(schemaPaths) == 0 {
		return nil, fmt.Errorf("no JSON schemas found")
	}

	for _, schemaPath := range schemaPaths {
		raw, err := fs.ReadFile(fsys, schemaPath)
		if err != nil {
			return nil, fmt.Errorf("read schema %s: %w", schemaPath, err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("parse schema %s: %w", schemaPath, err)
		}
		resourceURL := schemaBaseURL + path.Base(schemaPath)
		if err := compiler.AddResource(resourceURL, document); err != nil {
			return nil, fmt.Errorf("register schema %s: %w", schemaPath, err)
		}
	}

	return &Validator{
		fsys:     fsys,
		compiler: compiler,
		compiled: make(map[string]*jsonschema.Schema),
	}, nil
}

func (validator *Validator) ValidateFile(schemaName, instancePath string) error {
	if validator == nil {
		return fmt.Errorf("schema validator is required")
	}
	raw, err := fs.ReadFile(validator.fsys, instancePath)
	if err != nil {
		return fmt.Errorf("read schema instance %s: %w", instancePath, err)
	}
	if err := validator.Validate(schemaName, raw); err != nil {
		return fmt.Errorf("%s: %w", instancePath, err)
	}
	return nil
}

func (validator *Validator) Validate(schemaName string, raw []byte) error {
	if validator == nil || validator.compiler == nil {
		return fmt.Errorf("schema validator is not initialized")
	}
	if path.Base(schemaName) != schemaName || schemaName == "" {
		return fmt.Errorf("invalid schema name %q", schemaName)
	}

	schema, exists := validator.compiled[schemaName]
	if !exists {
		var err error
		schema, err = validator.compiler.Compile(schemaBaseURL + schemaName)
		if err != nil {
			return fmt.Errorf("compile schema %s: %w", schemaName, err)
		}
		validator.compiled[schemaName] = schema
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("parse JSON instance: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("does not satisfy %s: %w", schemaName, err)
	}
	return nil
}
