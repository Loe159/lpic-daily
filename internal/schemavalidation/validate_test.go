package schemavalidation_test

import (
	"testing"
	"testing/fstest"

	"github.com/Loe159/lpic-daily/internal/schemavalidation"
)

func TestValidatorAppliesDraft202012Constraints(t *testing.T) {
	fsys := fstest.MapFS{
		"schemas/common.schema.json": {
			Data: []byte(`{
				"$schema":"https://json-schema.org/draft/2020-12/schema",
				"$id":"https://lpic-daily.dev/schemas/common.schema.json",
				"$defs":{"name":{"type":"string","minLength":3}}
			}`),
		},
		"schemas/test.schema.json": {
			Data: []byte(`{
				"$schema":"https://json-schema.org/draft/2020-12/schema",
				"$id":"https://lpic-daily.dev/schemas/test.schema.json",
				"type":"object",
				"additionalProperties":false,
				"required":["name"],
				"properties":{"name":{"$ref":"common.schema.json#/$defs/name"}}
			}`),
		},
		"valid.json":   {Data: []byte(`{"name":"bash"}`)},
		"invalid.json": {Data: []byte(`{"name":"x","extra":true}`)},
	}

	validator, err := schemavalidation.New(fsys)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := validator.ValidateFile("test.schema.json", "valid.json"); err != nil {
		t.Fatalf("valid instance rejected: %v", err)
	}
	if err := validator.ValidateFile("test.schema.json", "invalid.json"); err == nil {
		t.Fatal("invalid instance unexpectedly accepted")
	}
}
