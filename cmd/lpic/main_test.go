package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
)

const (
	shellEnvironmentRepairID = "lpic1.103.1.shell-environment-repair"
	sharedDropboxID          = "lpic1.104.5.shared-dropbox"
	stuckWorkerID            = "lpic1.103.5.stuck-worker"
)

func TestTodayStartsWith1031AndCreatesLocalProgressStore(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	var stdout bytes.Buffer
	if err := runWithIO([]string{"today"}, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("today error = %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"LPIC Daily — Aujourd'hui",
		"Nouveau · 103.1 · syntaxe shell et séquences de commandes",
		"Cours conseillé: lpic1.103.1.lesson.shell-sequences",
		"lpic1.103.1.q.sequence-and",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("today output missing %q: %q", want, output)
		}
	}
}

func TestInitialAssessmentUnlocksDependentObjectiveWithoutLessonEvidence(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	input := strings.Join([]string{
		"&&",
		"export FOO=bar",
		"/opt/tools/tool",
		"simples",
		"history",
		"type",
		"uname -r",
		"",
	}, "\n")

	var stdout bytes.Buffer
	if err := runWithIO(
		[]string{"assess"},
		strings.NewReader(input),
		&stdout,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("assess error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Résultat: 7/7") ||
		!strings.Contains(stdout.String(), "Foundation 103.1 prête") {
		t.Fatalf("assessment output = %q", stdout.String())
	}

	ctx := context.Background()
	store, err := openProgressStore(ctx)
	if err != nil {
		t.Fatalf("open progress store = %v", err)
	}
	defer store.Close()

	curriculumBundle, err := curriculum.Load(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("curriculum.Load() error = %v", err)
	}
	for _, conceptID := range curriculumBundle.Phase1.ObjectiveConcepts["103.1"] {
		events, err := store.EvidenceForConcept(ctx, conceptID)
		if err != nil {
			t.Fatalf("EvidenceForConcept(%s) error = %v", conceptID, err)
		}
		if len(events) == 0 {
			t.Fatalf("no assessment evidence for %s", conceptID)
		}
		for _, event := range events {
			if event.ActivityKind == learning.ActivityLesson {
				t.Fatalf("assessment fabricated lesson evidence: %#v", event)
			}
		}
	}

	var today bytes.Buffer
	if err := runWithIO([]string{"today"}, strings.NewReader(""), &today, &bytes.Buffer{}); err != nil {
		t.Fatalf("today after assessment error = %v", err)
	}
	if !strings.Contains(today.String(), "103.5") {
		t.Fatalf("today did not unlock preferred dependent objective: %q", today.String())
	}
}

func TestLearnRecordsExposureAndAdvancesNewConcept(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	var lessonOut bytes.Buffer
	if err := runWithIO(
		[]string{"learn", "lpic1.103.1.lesson.shell-sequences"},
		strings.NewReader("o\n"),
		&lessonOut,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("learn error = %v", err)
	}
	if !strings.Contains(lessonOut.String(), "Progression enregistrée.") {
		t.Fatalf("learn output = %q", lessonOut.String())
	}

	var todayOut bytes.Buffer
	if err := runWithIO([]string{"today"}, strings.NewReader(""), &todayOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("today after lesson error = %v", err)
	}
	if !strings.Contains(todayOut.String(), "Consolidation · 103.1 · syntaxe shell et séquences de commandes") ||
		!strings.Contains(todayOut.String(), "lpic1.103.1.q.sequence-and") {
		t.Fatalf("today did not request immediate consolidation: %q", todayOut.String())
	}
}

func TestQuestionGradesNumberedChoiceAndRecordsResult(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	var stdout bytes.Buffer
	if err := runWithIO(
		[]string{"question", "lpic1.103.1.q.sequence-and"},
		strings.NewReader("2\n"),
		&stdout,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("question error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Correct.") {
		t.Fatalf("question output = %q", stdout.String())
	}
}

func TestQuestionRejectsUnknownChoiceWithoutRecording(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	var stdout bytes.Buffer
	err := runWithIO(
		[]string{"question", "lpic1.103.1.q.sequence-and"},
		strings.NewReader("99\n"),
		&stdout,
		&bytes.Buffer{},
	)
	if err == nil || !strings.Contains(err.Error(), "outside 1..3") {
		t.Fatalf("error = %v", err)
	}
}

func TestTodayRejectsUnknownOption(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	var stdout bytes.Buffer
	err := runWithIO([]string{"today", "--long"}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "usage: lpic today") {
		t.Fatalf("error = %v, want today usage error", err)
	}
}

func TestLabListAliasesAndShow(t *testing.T) {
	for _, args := range [][]string{{"lab", "list"}, {"labs"}} {
		var stdout bytes.Buffer
		if err := runWithIO(args, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
			t.Fatalf("%v error = %v", args, err)
		}
		text := stdout.String()
		for _, want := range []string{
			shellEnvironmentRepairID,
			sharedDropboxID,
			stuckWorkerID,
			"12 min",
			"15 min",
			"Réparer un environnement de login shell",
			"podman",
			"fedora",
			"Sécuriser un répertoire partagé",
			"Diagnostiquer des jobs",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%v output missing %q: %q", args, want, text)
			}
		}
	}

	var stdout bytes.Buffer
	if err := runWithIO(
		[]string{"lab", "show", sharedDropboxID},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("lab show error = %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"Critères de réussite", "groupe project", "héritent du groupe project", "4 niveaux d'indices"} {
		if !strings.Contains(output, want) {
			t.Fatalf("show output missing %q: %q", want, output)
		}
	}
	if strings.Contains(output, "3770") || strings.Contains(output, "chown root:project") {
		t.Fatalf("show leaked exact/reference solution: %q", output)
	}
}

func TestLabHintAndDebriefDisclosure(t *testing.T) {
	var first bytes.Buffer
	if err := runWithIO(
		[]string{"lab", "hint", sharedDropboxID, "1"},
		strings.NewReader(""),
		&first,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("hint 1 error = %v", err)
	}
	if strings.Contains(first.String(), "3770") {
		t.Fatalf("hint 1 leaked exact solution: %q", first.String())
	}

	var solution bytes.Buffer
	if err := runWithIO(
		[]string{"lab", "hint", sharedDropboxID, "4"},
		strings.NewReader(""),
		&solution,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("hint 4 error = %v", err)
	}
	if !strings.Contains(solution.String(), "3770") || !strings.Contains(solution.String(), "révèle la solution") {
		t.Fatalf("hint 4 output = %q", solution.String())
	}

	var debrief bytes.Buffer
	if err := runWithIO(
		[]string{"lab", "debrief", sharedDropboxID},
		strings.NewReader(""),
		&debrief,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("debrief error = %v", err)
	}
	if !strings.Contains(debrief.String(), "3770") || !strings.Contains(debrief.String(), "sticky bit") {
		t.Fatalf("debrief output = %q", debrief.String())
	}
}

func TestValidateIncludesLabs(t *testing.T) {
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	var stdout bytes.Buffer
	if err := runWithIO([]string{"validate"}, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("validate error = %v", err)
	}
	want := fmt.Sprintf("builtin labs OK: %d", len(labs))
	if !strings.Contains(stdout.String(), want) {
		t.Fatalf("validate output = %q, want %q", stdout.String(), want)
	}
}

func TestLabRunFailsClosedWhenRootlessPodmanIsUnavailable(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runWithIO(
		[]string{"lab", "run", sharedDropboxID},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if err == nil {
		t.Fatal("lab run unexpectedly succeeded without Podman")
	}
	if !strings.Contains(err.Error(), "open rootless Podman backend") {
		t.Fatalf("error = %v, want rootless Podman failure", err)
	}
	if strings.Contains(stdout.String(), "Lab réussi") {
		t.Fatalf("lab falsely reported success: %q", stdout.String())
	}
}

func TestUnknownLabFails(t *testing.T) {
	var stdout bytes.Buffer
	err := runWithIO(
		[]string{"lab", "show", "lpic1.999.9.missing"},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
	)
	if err == nil || !strings.Contains(err.Error(), "unknown lab") {
		t.Fatalf("error = %v, want unknown lab", err)
	}
}

func TestCorrectRecognitionAfterLessonAdvancesToNextConcept(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	if err := runWithIO(
		[]string{"learn", "lpic1.103.1.lesson.shell-sequences"},
		strings.NewReader("o\n"),
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("learn error = %v", err)
	}
	if err := runWithIO(
		[]string{"question", "lpic1.103.1.q.sequence-and"},
		strings.NewReader("2\n"),
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("question error = %v", err)
	}

	var todayOut bytes.Buffer
	if err := runWithIO([]string{"today"}, strings.NewReader(""), &todayOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("today error = %v", err)
	}
	if !strings.Contains(todayOut.String(), "Nouveau · 103.1 · variables shell/environnement") {
		t.Fatalf("today did not advance after consolidation: %q", todayOut.String())
	}
}
