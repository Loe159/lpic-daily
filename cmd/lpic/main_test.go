package main

import (
	"bytes"
	"strings"
	"testing"
)

const sharedDropboxID = "lpic1.104.5.shared-dropbox"

func TestValidateIncludesBuiltinLabs(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"validate"}, &output); err != nil {
		t.Fatalf("run(validate) error = %v", err)
	}
	if !strings.Contains(output.String(), "builtin labs OK: 1") {
		t.Fatalf("validate output = %q", output.String())
	}
}

func TestLabsListsSharedDropbox(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"labs"}, &output); err != nil {
		t.Fatalf("run(labs) error = %v", err)
	}
	text := output.String()
	for _, want := range []string{sharedDropboxID, "12 min", "podman", "fedora", "Sécuriser un répertoire partagé"} {
		if !strings.Contains(text, want) {
			t.Fatalf("labs output missing %q: %q", want, text)
		}
	}
}

func TestLabShowDoesNotRevealDebriefOrExactSolution(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"lab", "show", sharedDropboxID}, &output); err != nil {
		t.Fatalf("run(lab show) error = %v", err)
	}
	text := output.String()
	for _, want := range []string{
		"Critères de réussite",
		"groupe project",
		"héritent du groupe project",
		"4 niveaux d'indices",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("show output missing %q: %q", want, text)
		}
	}
	if strings.Contains(text, "3770") {
		t.Fatalf("show output leaked exact solution: %q", text)
	}
	if strings.Contains(text, "Solution de référence") {
		t.Fatalf("show output leaked reference solution: %q", text)
	}
}

func TestLabHintsAreExplicitlyProgressive(t *testing.T) {
	var first bytes.Buffer
	if err := run([]string{"lab", "hint", sharedDropboxID, "1"}, &first); err != nil {
		t.Fatalf("hint 1 error = %v", err)
	}
	if strings.Contains(first.String(), "3770") {
		t.Fatalf("hint 1 leaked exact solution: %q", first.String())
	}

	var solution bytes.Buffer
	if err := run([]string{"lab", "hint", sharedDropboxID, "4"}, &solution); err != nil {
		t.Fatalf("hint 4 error = %v", err)
	}
	if !strings.Contains(solution.String(), "3770") || !strings.Contains(solution.String(), "révèle la solution") {
		t.Fatalf("hint 4 should explicitly reveal/mark solution: %q", solution.String())
	}
}

func TestLabDebriefRequiresExplicitCommand(t *testing.T) {
	var output bytes.Buffer
	if err := run([]string{"lab", "debrief", sharedDropboxID}, &output); err != nil {
		t.Fatalf("debrief error = %v", err)
	}
	if !strings.Contains(output.String(), "3770") || !strings.Contains(output.String(), "sticky bit") {
		t.Fatalf("debrief output = %q", output.String())
	}
}

func TestUnknownLabFails(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"lab", "show", "lpic1.999.9.missing"}, &output)
	if err == nil || !strings.Contains(err.Error(), "unknown lab") {
		t.Fatalf("error = %v, want unknown lab", err)
	}
}
