package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	lpicdaily "github.com/Loe159/lpic-daily"
	"github.com/Loe159/lpic-daily/internal/curriculum"
	"github.com/Loe159/lpic-daily/internal/lab"
	"github.com/Loe159/lpic-daily/internal/learning"
	"github.com/Loe159/lpic-daily/internal/runner"
)

const (
	partitionFilesystemsID   = "lpic1.104.1.partition-filesystems"
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
		"lpic1.103.1.syntaxe-shell-et-sequences-de-commandes.q.autonomous-recall",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("today output missing %q: %q", want, output)
		}
	}
}

func TestInitialAssessmentMarks1031ReadyWithoutLessonEvidence(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	input := strings.Join([]string{
		"&&",
		"export FOO=bar",
		"/opt/tools/tool",
		"simples",
		"history",
		"type",
		"uname -r",
		"env",
		"export",
		"PATH",
		".bash_history",
		"echo",
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
	if !strings.Contains(stdout.String(), "Résultat: 12/12") ||
		!strings.Contains(stdout.String(), "Rappel 103.1 solide") {
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
		!strings.Contains(todayOut.String(), "lpic1.103.1.syntaxe-shell-et-sequences-de-commandes.q.autonomous-recall") {
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
	output := stdout.String()
	for _, want := range []string{"✓ CORRECT", "`&&` exécute la commande de droite seulement si la commande de gauche retourne le statut 0.", "XP:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("question output missing %q: %q", want, output)
		}
	}
	if strings.Index(output, "✓ CORRECT") > strings.Index(output, "XP:") {
		t.Fatalf("question result should be visible before gamification output: %q", output)
	}
}

func TestQuestionIncorrectFeedbackIsVisuallyDistinct(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	var stdout bytes.Buffer
	if err := runWithIO(
		[]string{"question", "lpic1.103.1.q.single-quotes"},
		strings.NewReader("2\n"),
		&stdout,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("question error = %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"✗ INCORRECT",
		"Les apostrophes simples désactivent l'expansion des paramètres dans leur contenu.",
		"Cette notion reste à consolider.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("question output missing %q: %q", want, output)
		}
	}
	if strings.Index(output, "✗ INCORRECT") > strings.Index(output, "XP:") {
		t.Fatalf("question result should be visible before gamification output: %q", output)
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
	for _, want := range []string{"Réussite", "groupe project", "conserve le groupe project", "Commandes utiles"} {
		if !strings.Contains(output, want) {
			t.Fatalf("show output missing %q: %q", want, output)
		}
	}
	if strings.Contains(output, "3770") || strings.Contains(output, "chown root:project") {
		t.Fatalf("show leaked exact/reference solution: %q", output)
	}
}

func TestLabHintAndDebriefDisclosure(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
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
	if !strings.Contains(err.Error(), "prepare Podman lab environment") &&
		!strings.Contains(err.Error(), "open rootless Podman backend") {
		t.Fatalf("error = %v, want fail-closed Podman environment failure", err)
	}
	if strings.Contains(stdout.String(), "Lab réussi") {
		t.Fatalf("lab falsely reported success: %q", stdout.String())
	}
}

func TestPrepareJobControlShellAvoidsUnsupportedNonTTYStdin(t *testing.T) {
	fake := &scriptedLabRunner{rejectNonTTYStdin: true}
	if err := prepareJobControlShell(context.Background(), fake, runner.Instance{ID: "scripted"}); err != nil {
		t.Fatalf("prepareJobControlShell() error = %v", err)
	}
	if len(fake.execRequests) != 2 {
		t.Fatalf("exec requests = %d, want 2", len(fake.execRequests))
	}
	for index, request := range fake.execRequests {
		if request.Stdin != nil {
			t.Fatalf("request %d unexpectedly uses stdin", index+1)
		}
		if request.TTY {
			t.Fatalf("request %d unexpectedly uses TTY", index+1)
		}
		if len(request.Argv) < 6 || request.Argv[0] != "/usr/bin/bash" {
			t.Fatalf("request %d argv = %#v, want structured bash file write", index+1, request.Argv)
		}
	}
}

func TestInteractiveTerminalRejectsBufferedIO(t *testing.T) {
	if interactiveTerminal(strings.NewReader(""), &bytes.Buffer{}) {
		t.Fatal("buffered I/O must not be treated as an interactive terminal")
	}
}

func TestSanitizedTerminalWriterRemovesTerminalControls(t *testing.T) {
	var output bytes.Buffer
	writer := sanitizedTerminalWriter{destination: &output}

	payload := []byte("safe\x1b[31mred\x1b[0m\rrewrite\x07\u009b31m\n")
	n, err := writer.Write(payload)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(payload) {
		t.Fatalf("Write() count = %d, want %d", n, len(payload))
	}
	got := output.String()
	for _, forbidden := range []rune{'\x1b', '\r', '\x07', '\u009b'} {
		if strings.ContainsRune(got, forbidden) {
			t.Fatalf("sanitized output still contains control %U: %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "safe") || !strings.Contains(got, "red") {
		t.Fatalf("sanitized output lost printable content: %q", got)
	}
}

func TestRequireUnprivilegedVMProcessRejectsRoot(t *testing.T) {
	if err := requireUnprivilegedVMProcess(0); err == nil || !strings.Contains(err.Error(), "not root") {
		t.Fatalf("root process error = %v", err)
	}
	if err := requireUnprivilegedVMProcess(1000); err != nil {
		t.Fatalf("regular user rejected: %v", err)
	}
}

func TestLibvirtLabRunFailsClosedWithoutReadyEnvironment(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LPIC_DAILY_VM_IMAGE_DIR", filepath.Join(root, "vm-images"))
	t.Setenv("LPIC_DAILY_STATE_DIR", filepath.Join(root, "state"))

	var stdout bytes.Buffer
	err := runWithIO(
		[]string{"lab", "run", partitionFilesystemsID},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("libvirt lab unexpectedly started")
	}
	if os.Geteuid() == 0 {
		if !strings.Contains(err.Error(), "regular user, not root") {
			t.Fatalf("error = %v, want root refusal", err)
		}
	} else if !strings.Contains(err.Error(), "prepare VM lab environment") &&
		!strings.Contains(err.Error(), "open libvirt") {
		t.Fatalf("error = %v, want fail-closed VM environment failure", err)
	}
	if strings.Contains(err.Error(), "Podman") {
		t.Fatalf("libvirt lab unexpectedly fell back to Podman: %v", err)
	}
	if strings.Contains(stdout.String(), "Lab réussi") {
		t.Fatalf("libvirt lab falsely reported success: %q", stdout.String())
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

func TestCorrectRecognitionAfterLessonRequiresPracticalConsolidation(t *testing.T) {
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
	if !strings.Contains(todayOut.String(), "Consolidation · 103.1 · syntaxe shell et séquences de commandes") ||
		!strings.Contains(todayOut.String(), "Lab: lpic1.103.1.shell-environment-repair") {
		t.Fatalf("today did not require practical consolidation before advancing: %q", todayOut.String())
	}
}

func TestLabResetDoesNotEraseSolutionRevealEvidence(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	var authored lab.Lab
	for _, candidate := range labs {
		if candidate.Definition.ID == sharedDropboxID {
			authored = candidate
			break
		}
	}
	if authored.Definition.ID == "" {
		t.Fatal("shared-dropbox lab not found")
	}

	input := strings.NewReader(":hint\n:hint\n:hint\n:hint\n:reset\n:check\n")
	var stdout, stderr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(), authored, &scriptedLabRunner{}, false,
		input, &stdout, &stderr,
	); err != nil {
		t.Fatalf("runInteractiveLabWithBackend() error = %v; stderr=%q", err, stderr.String())
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	defer store.Close()

	events, err := store.EvidenceForConcept(context.Background(), authored.Definition.ConceptIDs[0])
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].HighestHintLevel != 4 || !events[0].SolutionRevealed {
		t.Fatalf("reset laundered hint evidence: %#v", events[0])
	}
	projection, err := learning.ProjectMastery(events[0].ConceptID, events, learning.DefaultProjectionPolicy())
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if projection.Stage != learning.StageGuided {
		t.Fatalf("stage = %s, want guided after solution reveal", projection.Stage)
	}
}

type scriptedLabRunner struct {
	failChecks        bool
	rejectNonTTYStdin bool
	execRequests      []runner.ExecRequest
}

func (*scriptedLabRunner) Prepare(context.Context, runner.Definition) (runner.Instance, error) {
	return runner.Instance{ID: "scripted"}, nil
}
func (*scriptedLabRunner) Start(context.Context, runner.Instance) error { return nil }
func (fake *scriptedLabRunner) Exec(_ context.Context, _ runner.Instance, request runner.ExecRequest) (runner.ExecResult, error) {
	if fake.rejectNonTTYStdin && request.Stdin != nil && !request.TTY {
		return runner.ExecResult{}, errors.New("non-TTY stdin rejected")
	}
	fake.execRequests = append(fake.execRequests, request)
	return runner.ExecResult{ExitCode: 0}, nil
}
func (fake *scriptedLabRunner) Stat(_ context.Context, _ runner.Instance, guestPath string) (runner.FileInfo, error) {
	if guestPath == "/srv/shared/team-note" {
		return runner.FileInfo{
			Path: guestPath, Mode: 0o0660, UID: 0, GID: 2000,
			User: "root", Group: "project", IsDir: false,
		}, nil
	}
	if guestPath == "/srv/shared/audit-helper" {
		return runner.FileInfo{
			Path: guestPath, Mode: 0o0755, UID: 0, GID: 0,
			User: "root", Group: "root", IsDir: false,
		}, nil
	}
	mode := uint32(0o3770)
	if fake.failChecks {
		mode = 0o0770
	}
	return runner.FileInfo{Path: guestPath, Mode: mode, UID: 0, GID: 2000, User: "root", Group: "project", IsDir: true}, nil
}
func (*scriptedLabRunner) ReadFile(context.Context, runner.Instance, string, int64) ([]byte, error) {
	return nil, errors.New("unexpected ReadFile")
}
func (*scriptedLabRunner) Processes(context.Context, runner.Instance) ([]runner.Process, error) {
	return nil, nil
}
func (*scriptedLabRunner) Reset(context.Context, runner.Instance) error   { return nil }
func (*scriptedLabRunner) Destroy(context.Context, runner.Instance) error { return nil }

type disclosureInjectingRunner struct {
	scriptedLabRunner
	labID    string
	injected bool
}

func (fake *disclosureInjectingRunner) Stat(
	ctx context.Context,
	instance runner.Instance,
	guestPath string,
) (runner.FileInfo, error) {
	if !fake.injected {
		fake.injected = true
		if err := recordLabDisclosure(ctx, fake.labID, 4, true); err != nil {
			return runner.FileInfo{}, err
		}
	}
	return fake.scriptedLabRunner.Stat(ctx, instance, guestPath)
}

func TestStandaloneSolutionHintTaintsNextSuccessfulLabAttempt(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	if err := runWithIO(
		[]string{"lab", "hint", sharedDropboxID, "4"},
		strings.NewReader(""),
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("standalone hint error = %v", err)
	}

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, sharedDropboxID)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		&scriptedLabRunner{},
		false,
		strings.NewReader(":check\n"),
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("runInteractiveLabWithBackend() error = %v; stderr=%q", err, stderr.String())
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	defer store.Close()

	events, err := store.EvidenceForConcept(context.Background(), authored.Definition.ConceptIDs[0])
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].HighestHintLevel != 4 || !events[0].SolutionRevealed {
		t.Fatalf("standalone hint was laundered: %#v", events[0])
	}
	projection, err := learning.ProjectMastery(
		events[0].ConceptID,
		events,
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("ProjectMastery() error = %v", err)
	}
	if projection.Stage != learning.StageGuided {
		t.Fatalf("stage = %s, want guided after standalone solution reveal", projection.Stage)
	}

	disclosure, err := store.LabDisclosure(context.Background(), authored.Definition.ID)
	if err != nil {
		t.Fatalf("LabDisclosure() error = %v", err)
	}
	if disclosure.HighestHintLevel != 0 {
		t.Fatalf("successful lab did not clear disclosure: %#v", disclosure)
	}
}

func TestConcurrentSolutionDisclosureTaintsActiveLabAttempt(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, sharedDropboxID)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}

	fake := &disclosureInjectingRunner{labID: authored.Definition.ID}
	var stdout, stderr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		fake,
		false,
		strings.NewReader(":check\n"),
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("runInteractiveLabWithBackend() error = %v; stderr=%q", err, stderr.String())
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	defer store.Close()
	events, err := store.EvidenceForConcept(context.Background(), authored.Definition.ConceptIDs[0])
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(events) != 1 || events[0].HighestHintLevel != 4 || !events[0].SolutionRevealed {
		t.Fatalf("concurrent disclosure was laundered: %#v", events)
	}
}

func TestInteractiveSolutionHintPersistsAcrossRestart(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, sharedDropboxID)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}

	var firstOut, firstErr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		&scriptedLabRunner{},
		false,
		strings.NewReader(":hint\n:hint\n:hint\n:hint\n:quit\n"),
		&firstOut,
		&firstErr,
	); err != nil {
		t.Fatalf("first lab run error = %v; stderr=%q", err, firstErr.String())
	}

	var secondOut, secondErr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		&scriptedLabRunner{},
		false,
		strings.NewReader(":check\n"),
		&secondOut,
		&secondErr,
	); err != nil {
		t.Fatalf("second lab run error = %v; stderr=%q", err, secondErr.String())
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	defer store.Close()
	events, err := store.EvidenceForConcept(context.Background(), authored.Definition.ConceptIDs[0])
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(events) != 1 || events[0].HighestHintLevel != 4 || !events[0].SolutionRevealed {
		t.Fatalf("restart laundered interactive hint: %#v", events)
	}
}

func TestFailedLabCheckRecordsConceptGranularEvidence(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, sharedDropboxID)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		&scriptedLabRunner{failChecks: true},
		false,
		strings.NewReader(":check\n:quit\n"),
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("runInteractiveLabWithBackend() error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Tentative enregistrée.") {
		t.Fatalf("failed attempt was not acknowledged: %q", stdout.String())
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	defer store.Close()

	assertResult := func(conceptID string, want learning.Result) learning.EvidenceEvent {
		t.Helper()
		events, err := store.EvidenceForConcept(context.Background(), conceptID)
		if err != nil {
			t.Fatalf("EvidenceForConcept(%s) error = %v", conceptID, err)
		}
		if len(events) != 1 || events[0].Result != want {
			t.Fatalf("events for %s = %#v, want one %s result", conceptID, events, want)
		}
		return events[0]
	}

	permission := assertResult("lpic1.104.5.permissions-rwx-fichier-dossier", learning.ResultPass)
	sticky := assertResult("lpic1.104.5.sticky-bit", learning.ResultPartial)

	permissionProjection, err := learning.ProjectMastery(
		permission.ConceptID,
		[]learning.EvidenceEvent{permission},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("permission ProjectMastery() error = %v", err)
	}
	if permissionProjection.Stage != learning.StageIndependent {
		t.Fatalf("permission projection = %#v, want independent", permissionProjection)
	}

	stickyProjection, err := learning.ProjectMastery(
		sticky.ConceptID,
		[]learning.EvidenceEvent{sticky},
		learning.DefaultProjectionPolicy(),
	)
	if err != nil {
		t.Fatalf("sticky ProjectMastery() error = %v", err)
	}
	if stickyProjection.Stage != learning.StageUnseen || stickyProjection.Partials != 1 {
		t.Fatalf("sticky projection = %#v, want unseen with one partial", stickyProjection)
	}
}

func TestPersistentShellRequirementBlocksStateOnlySuccess(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, sharedDropboxID)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}
	authored.Definition.NeedsPersistentShell = true

	var stdout, stderr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		&scriptedLabRunner{},
		true,
		strings.NewReader(":check\n:quit\n"),
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("runInteractiveLabWithBackend() error = %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".persistent-shell") {
		t.Fatalf("missing persistent-shell rejection: %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Lab réussi.") {
		t.Fatalf("lab succeeded without persistent shell: %q", stdout.String())
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("open progress store: %v", err)
	}
	defer store.Close()

	conceptID := authored.Definition.ConceptIDs[0]
	events, err := store.EvidenceForConcept(context.Background(), conceptID)
	if err != nil {
		t.Fatalf("EvidenceForConcept() error = %v", err)
	}
	if len(events) != 1 || events[0].Result != learning.ResultFail {
		t.Fatalf("events = %#v, want one failed attempt", events)
	}
}

func TestPersistentShellFailureTargetsJobControlConcept(t *testing.T) {
	definition := lab.Definition{
		ConceptIDs: []string{
			"lpic1.103.5.jobs-du-shell",
			"lpic1.103.5.signaux",
		},
	}
	results := map[string]learning.Result{
		"lpic1.103.5.jobs-du-shell": learning.ResultPass,
		"lpic1.103.5.signaux":       learning.ResultPass,
	}

	markPersistentShellConceptFailure(definition, results)

	if got := results["lpic1.103.5.jobs-du-shell"]; got != learning.ResultFail {
		t.Fatalf("job-control result = %s, want fail", got)
	}
	if got := results["lpic1.103.5.signaux"]; got != learning.ResultPass {
		t.Fatalf("unrelated signal result = %s, want pass", got)
	}
}

func TestJobControlEvidenceRequiresCtrlZJobsAndSuccessfulBg(t *testing.T) {
	evidence := &jobControlInteractionEvidence{}
	evidence.observeShellEvents([]byte("jobs\n"))
	evidence.sawCtrlZ = true
	if evidence.Complete() {
		t.Fatal("Ctrl-Z plus jobs must not pass without a successful bg")
	}

	evidence.observeShellEvents([]byte("bg\n"))
	if !evidence.Complete() {
		t.Fatal("Ctrl-Z plus jobs plus successful bg should satisfy job-control evidence")
	}

	evidence.Reset()
	if evidence.Complete() {
		t.Fatal("Reset() must clear job-control evidence")
	}
}

func TestLabRequiresJobControlOnlyForJobConcept(t *testing.T) {
	if !labRequiresJobControl(lab.Definition{ConceptIDs: []string{"lpic1.103.5.jobs-du-shell"}}) {
		t.Fatal("jobs-du-shell should require job-control evidence")
	}
	if labRequiresJobControl(lab.Definition{ConceptIDs: []string{"lpic1.103.5.signaux"}}) {
		t.Fatal("unrelated signal concept should not require job-control evidence")
	}
}

func TestVMConsoleEscapeReaderStopsAtControlRightBracket(t *testing.T) {
	escapeCalled := false
	reader := &vmConsoleEscapeReader{
		reader: strings.NewReader("before\x1dafter"),
		onEscape: func() {
			escapeCalled = true
		},
	}
	buffer := make([]byte, 64)

	n, err := reader.Read(buffer)
	if err != nil {
		t.Fatalf("first Read() error = %v", err)
	}
	if got := string(buffer[:n]); got != "before" {
		t.Fatalf("first Read() = %q, want %q", got, "before")
	}
	if !escapeCalled {
		t.Fatal("Ctrl-] did not trigger console cancellation")
	}

	n, err = reader.Read(buffer)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("second Read() = (%d, %v), want (0, EOF)", n, err)
	}
}

func TestVMConsoleEscapeReaderHandlesImmediateEscape(t *testing.T) {
	reader := &vmConsoleEscapeReader{reader: strings.NewReader("\x1dignored")}
	buffer := make([]byte, 64)

	n, err := reader.Read(buffer)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read() = (%d, %v), want (0, EOF)", n, err)
	}
}

type consoleScriptedLabRunner struct {
	scriptedLabRunner
	consoleCalls int
}

func (fake *consoleScriptedLabRunner) OpenConsole(
	context.Context,
	runner.Instance,
	runner.ConsoleRequest,
) error {
	fake.consoleCalls++
	return nil
}

type rebootScriptedLabRunner struct {
	scriptedLabRunner
	rebootCalls int
}

func (fake *rebootScriptedLabRunner) Reboot(context.Context, runner.Instance) error {
	fake.rebootCalls++
	return nil
}

func TestRunVMConsoleRejectsNonTerminalBeforeOpeningConsole(t *testing.T) {
	fake := &consoleScriptedLabRunner{}
	err := runVMConsole(
		context.Background(),
		fake,
		runner.Instance{ID: "vm"},
		strings.NewReader(""),
		&bytes.Buffer{},
	)
	if err == nil || !strings.Contains(err.Error(), "interactive terminal on stdin") {
		t.Fatalf("runVMConsole() error = %v", err)
	}
	if fake.consoleCalls != 0 {
		t.Fatalf("console calls = %d, want 0", fake.consoleCalls)
	}
}

func TestInteractiveVMLabDispatchesConsoleCommand(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, "lpic1.102.2.grub-kernel-parameter")
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}
	fake := &consoleScriptedLabRunner{}
	err = runInteractiveLabWithBackend(
		context.Background(),
		authored,
		fake,
		false,
		strings.NewReader(":console\n"),
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	if err == nil || !strings.Contains(err.Error(), "interactive terminal on stdin") {
		t.Fatalf(":console error = %v", err)
	}
}

func TestInteractiveVMLabDispatchesRebootCommand(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())
	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, "lpic1.102.2.grub-kernel-parameter")
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}
	fake := &rebootScriptedLabRunner{}
	var stdout, stderr bytes.Buffer
	if err := runInteractiveLabWithBackend(
		context.Background(),
		authored,
		fake,
		false,
		strings.NewReader(":reboot\n:quit\n"),
		&stdout,
		&stderr,
	); err != nil {
		t.Fatalf("runInteractiveLabWithBackend() error = %v; stderr=%q", err, stderr.String())
	}
	if fake.rebootCalls != 1 {
		t.Fatalf("reboot calls = %d, want 1", fake.rebootCalls)
	}
	if !strings.Contains(stdout.String(), "Reboot demandé") {
		t.Fatalf("stdout missing reboot confirmation: %q", stdout.String())
	}
}

func TestUpdateHelpDoesNotRunUpdater(t *testing.T) {
	var stdout bytes.Buffer
	if err := runWithIO([]string{"update", "--help"}, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("update --help error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage: lpic update") {
		t.Fatalf("update help output = %q", stdout.String())
	}
}


func TestDashboardLabParentRejectsCheckAndWaitsForRecordedSuccess(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(
		labs,
		"lpic1.103.1.set-env-et-portee-des-variables.standalone-diagnostic",
	)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("openProgressStore() error = %v", err)
	}
	if err := study.RecordLab(context.Background(), store, authored, 0, time.Now()); err != nil {
		_ = store.Close()
		t.Fatalf("RecordLab() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close progress store: %v", err)
	}

	var stdout bytes.Buffer
	if err := waitForDashboardLabCompletion(
		context.Background(),
		authored,
		0,
		strings.NewReader(":check\n\n"),
		&stdout,
	); err != nil {
		t.Fatalf("waitForDashboardLabCompletion() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, ":check doit être saisi dans le terminal enfant") {
		t.Fatalf("parent did not reject :check: %q", output)
	}
	if !strings.Contains(output, "Progression du lab détectée.") {
		t.Fatalf("recorded success was not detected: %q", output)
	}
}

func TestSuccessfulLabAttemptCountRequiresEveryMappedConcept(t *testing.T) {
	t.Setenv("LPIC_DAILY_STATE_DIR", t.TempDir())

	labs, err := lab.LoadAll(lpicdaily.BuiltinFS)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	authored, err := findLab(labs, shellEnvironmentRepairID)
	if err != nil {
		t.Fatalf("findLab() error = %v", err)
	}

	store, err := openProgressStore(context.Background())
	if err != nil {
		t.Fatalf("openProgressStore() error = %v", err)
	}
	at := time.Now()
	event := learning.EvidenceEvent{
		EventID:          "partial-lab-success",
		OccurredAt:       at,
		ConceptID:        authored.Definition.ConceptIDs[0],
		ObjectiveIDs:     append([]string(nil), authored.Definition.ObjectiveIDs...),
		SourceItemID:     authored.Definition.ID,
		ActivityKind:     learning.ActivityLab,
		EvidenceKind:     learning.EvidenceIndependentPractice,
		Result:           learning.ResultPass,
		Distribution:     authored.Definition.Environment.Distribution,
		PracticeContext:  authored.Definition.PracticeContext,
		AttemptIndex:     1,
	}
	if err := store.AppendEvidence(context.Background(), event); err != nil {
		_ = store.Close()
		t.Fatalf("AppendEvidence() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close progress store: %v", err)
	}

	count, err := successfulLabAttemptCount(context.Background(), authored)
	if err != nil {
		t.Fatalf("successfulLabAttemptCount() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("successfulLabAttemptCount() = %d, want 0 for a partial concept pass", count)
	}
}
