package curriculum

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
)

const (
	objectivesPath    = "curriculum/lpic-1-v5/objectives.json"
	prerequisitesPath = "curriculum/lpic-1-v5/prerequisites.json"
	conceptsPath      = "curriculum/lpic-1-v5/concepts.json"
	phase1Path        = "curriculum/lpic-1-v5/phase1-slice.json"
	schemasDir        = "schemas"
)

func Load(fsys fs.FS) (*Bundle, error) {
	var bundle Bundle

	if err := decodeStrictFile(fsys, objectivesPath, &bundle.Objectives); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, prerequisitesPath, &bundle.Prerequisites); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, conceptsPath, &bundle.Concepts); err != nil {
		return nil, err
	}
	if err := decodeStrictFile(fsys, phase1Path, &bundle.Phase1); err != nil {
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
