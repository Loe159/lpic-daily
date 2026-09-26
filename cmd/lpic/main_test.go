package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLabListAndShow(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := runWithIO([]string{"lab", "list"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("lab list error = %v", err)
	}
	if !strings.Contains(stdout.String(), "lpic1.104.5.shared-dropbox") {
		t.Fatalf("lab list output = %q", stdout.String())
	}

	stdout.Reset()
	if err := runWithIO(
		[]string{"lab", "show", "lpic1.104.5.shared-dropbox"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("lab show error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Sécuriser un répertoire partagé") {
		t.Fatalf("show output = %q", output)
	}
	if strings.Contains(output, "chmod 3770") || strings.Contains(output, "chown root:project") {
		t.Fatalf("show leaked reference solution: %q", output)
	}
}

func TestValidateIncludesLabs(t *testing.T) {
	var stdout bytes.Buffer
	if err := runWithIO([]string{"validate"}, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("validate error = %v", err)
	}
	if !strings.Contains(stdout.String(), "builtin labs: 1 OK") {
		t.Fatalf("validate output = %q", stdout.String())
	}
}
