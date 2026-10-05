package curriculum

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/Loe159/lpic-daily/internal/schemavalidation"
)

const (
	objectivesPath    = "curriculum/lpic-1-v5/objectives.json"
	prerequisitesPath = "curriculum/lpic-1-v5/prerequisites.json"
	conceptsPath      = "curriculum/lpic-1-v5/concepts.json"
	phase1Path        = "curriculum/lpic-1-v5/phase1-slice.json"
	phase3Path        = "curriculum/lpic-1-v5/phase3-exam101.json"
	phase4Path        = "curriculum/lpic-1-v5/phase4-exam102.json"
	schemasDir        = "schemas"
)

func Load(fsys fs.FS) (*Bundle, error) {
	var bundle Bundle

	schemaValidator, err := schemavalidation.New(fsys)
	if err != nil {
		return nil, fmt.Errorf("compile curriculum schemas: %w", err)
	}

	if err := decodeStrictFile(fsys, objectivesPath, &bundle.Objectives); err != nil {
		return nil, err
	}
	if err := schemaValidator.ValidateFile("objective-graph.schema.json", prerequisitesPath); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, prerequisitesPath, &bundle.Prerequisites); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, conceptsPath, &bundle.Concepts); err != nil {
		return nil, err
	}
	if err := validateConceptInstances(fsys, schemaValidator); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, phase1Path, &bundle.Phase1); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, phase3Path, &bundle.Phase3); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, phase4Path, &bundle.Phase4); err != nil {
		return nil, err
	}

	entries, err := fs.ReadDir(fsys, schemasDir)
	if err != nil {
		return nil, fmt.Errorf("read schemas directory: %w", err)
	}

	bundle.Schemas = make(map[string]map[string]any)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}
		var schema map[string]any
		schemaPath := path.Join(schemasDir, entry.Name())
		if err := decodeStrictFile(fsys, schemaPath, &schema); err != nil {
			return nil, err
		}
		bundle.Schemas[entry.Name()] = schema
	}

	if err := Validate(&bundle); err != nil {
		return nil, err
	}
	return &bundle, nil
}

func decodeStrictFile(fsys fs.FS, name string, out any) error {
	file, err := fsys.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode %s: trailing JSON value", name)
		}
		return fmt.Errorf("decode %s trailing data: %w", name, err)
	}
	return nil
}

func validateConceptInstances(fsys fs.FS, schemaValidator *schemavalidation.Validator) error {
	raw, err := fs.ReadFile(fsys, conceptsPath)
	if err != nil {
		return fmt.Errorf("read %s for schema validation: %w", conceptsPath, err)
	}
	var envelope struct {
		Concepts []json.RawMessage `json:"concepts"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode %s for schema validation: %w", conceptsPath, err)
	}
	for index, concept := range envelope.Concepts {
		if err := schemaValidator.Validate("concept.schema.json", concept); err != nil {
			return fmt.Errorf("%s concept %d: %w", conceptsPath, index+1, err)
		}
	}
	return nil
}
