package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Loe159/lpic-daily/internal/gamification"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/study"
)

func TestCheckResultsPrioritizeActionableFailures(t *testing.T) {
	var output bytes.Buffer
	PrintCheckResults(&output, []CheckPresentation{
		{Label: "Configuration déjà correcte", Passed: true},
		{Label: "Permissions du fichier", Diagnostic: "Mode incorrect"},
		{Label: "Service actif", Passed: true},
	})
	got := output.String()
	for _, want := range []string{"BILAN  2/3 OK", "1 à corriger", "À CORRIGER", "Permissions du fichier", "Mode incorrect", "  OK"} {
		if !strings.Contains(got, want) {
			t.Fatalf("check report missing %q: %q", want, got)
		}
	}
	if strings.Index(got, "Permissions du fichier") > strings.Index(got, "Configuration déjà correcte") {
		t.Fatalf("successful check is displayed ahead of the failure: %q", got)
	}
	if strings.Contains(got, "\x1b") {
		t.Fatalf("buffer output should not contain terminal escapes: %q", got)
	}
}

func TestCheckResultsCollapseManySuccesses(t *testing.T) {
	var output bytes.Buffer
	forChecks := make([]CheckPresentation, 6)
	for n := range forChecks {
		forChecks[n] = CheckPresentation{Label: "Critère validé", Passed: true}
	}
	PrintCheckResults(&output, forChecks)
	if !strings.Contains(output.String(), "BILAN  6/6 OK") || !strings.Contains(output.String(), "+ 4 autres critères OK") {
		t.Fatalf("missing compact summary: %q", output.String())
	}
}

func TestLabBriefUsesScannableSections(t *testing.T) {
	var output bytes.Buffer
	PrintLabBrief(&output, LabPresentation{
		Title:      "Rétablir le diagnostic",
		Objectives: []string{"103.1"},
		Minutes:    8,
		Mission:    "Un outil ne fonctionne plus. Rétablis son exécution sans modifier la politique.",
		Criteria:   []string{"Le rapport est accessible.", "La configuration est conservée."},
	})
	got := output.String()
	for _, want := range []string{"MISSION", "RÉUSSITE", "Rétablir le diagnostic", ":check vérifier", "□ Le rapport est accessible."} {
		if !strings.Contains(got, want) {
			t.Fatalf("brief missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Fatalf("output to pipe must not contain ANSI sequences: %q", got)
	}
}

func TestDashboardHidesTechnicalActivityIDs(t *testing.T) {
	d := NewDashboard(study.Plan{Items: []study.Item{{
		ConceptTitleFR:      "Localiser les outils",
		ObjectiveID:         "103.1",
		Kind:                learning.SessionNew,
		RecommendedLessonID: "lpic1.103.1.a-technical-slug",
		RecommendedLabID:    "lpic1.103.1.external-command-recovery",
	}}}, gamification.Snapshot{})
	d.height = 12
	d.width = 44
	got := d.render()
	for _, hidden := range []string{"a-technical-slug", "external-command-recovery", "\x1b"} {
		if strings.Contains(got, hidden) {
			t.Fatalf("dashboard leaked technical/UI data %q: %q", hidden, got)
		}
	}
	for _, want := range []string{"Entrée", "À FAIRE", "103.1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("dashboard missing %q: %q", want, got)
		}
	}
}

func TestTerminalColorsRespectNoColorFlag(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if Paint(ColorEnabled(&bytes.Buffer{}), ToneWarning, "À CORRIGER") != "À CORRIGER" {
		t.Fatal("unexpected ANSI styling")
	}
}
